package main

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"
)

const (
	LOGIN_HEADER = "api-token"
	AUTH_HEADER  = "x-auth-token"
)

type FlashBladeApi struct {
	restClient resty.Client
	apiToken   string
	apiVersion string
}

type ApiLoginResponse struct {
	Username string `json:"username"`
}

type ApiVolume struct {
	Destroyed        bool                      `json:"destroyed"`
	HardLimitEnabled bool                      `json:"hard_limit_enabled"`
	Name             string                    `json:"name"`
	Nfs              *ApiCreateVolumeNfsConfig `json:"nfs"`
	SizeInBytes      int64                     `json:"provisioned"`
}

type ApiGetVolumeResponse struct {
	Items          []ApiVolume `json:"items"`
	TotalItemCount int         `json:"total_item_count"`
}

type ApiCreateVolumeRequest struct {
	HardLimitEnabled bool                      `json:"hard_limit_enabled"`
	Nfs              *ApiCreateVolumeNfsConfig `json:"nfs"`
	SizeInBytes      int64                     `json:"provisioned"`
}

type ApiDestroyVolumeRequest struct {
	Destroyed bool                      `json:"destroyed"`
	Nfs       *ApiCreateVolumeNfsConfig `json:"nfs"`
}

type ApiCreateVolumeNfsConfig struct {
	V4_1_Enabled bool `json:"v4_1_enabled"`
}

type ApiCreateVolumeResponse struct {
	Items []ApiVolume `json:"items"`
}

func NewFlashBladeApi(apiUrl string, apiToken string, skipTlsVerification bool) *FlashBladeApi {
	if apiUrl == "" || apiToken == "" {
		klog.Errorf("apiUrl and apiToken must be set")
	}

	r := resty.New().
		SetBaseURL(apiUrl).
		SetHeader("Accept", "application/json").
		SetTLSClientConfig(&tls.Config{InsecureSkipVerify: skipTlsVerification})

	return &FlashBladeApi{
		restClient: *r,
		apiToken:   apiToken,
		apiVersion: "2.2",
	}
}

func (a *FlashBladeApi) SetDebug(debug bool) {
	a.restClient.SetDebug(true)
}

func (a *FlashBladeApi) login() error {
	klog.Infof("Logging in to FlashBlade API")

	resp, err := a.restClient.R().
		SetHeader(LOGIN_HEADER, a.apiToken).
		SetHeader(AUTH_HEADER, "").
		SetResult(&ApiLoginResponse{}).
		Post("api/login")

	if err != nil {
		klog.Errorf("Error during API login: %s", err)
		return err
	}

	authToken := resp.Header().Get(AUTH_HEADER)
	if authToken == "" {
		return errors.New("Auth token not returned by API")
	}
	a.restClient.SetHeader(AUTH_HEADER, authToken)

	loggedInUser := resp.Result().(*ApiLoginResponse)
	klog.Infof("Logged in successfully as %s", loggedInUser.Username)

	return nil
}

func (a *FlashBladeApi) apiBasePath() string {
	return "api/" + a.apiVersion
}

func (a *FlashBladeApi) FindVolumeByName(name string) (*ApiVolume, error) {
	loginErr := a.login()
	if loginErr != nil {
		return nil, loginErr
	}

	klog.Infof("Looking for volume with name %s", name)
	requestPath := fmt.Sprintf("%s/file-systems?offset=0&start=0&limit=1&sort=name&filter=name='%s'", a.apiBasePath(), name)

	resp, err := a.restClient.R().
		SetResult(&ApiGetVolumeResponse{}).
		Get(requestPath)
	if err != nil {
		klog.Errorf("Getting volume infos failed for volume: %s", name)
		return nil, err
	}

	volumes := resp.Result().(*ApiGetVolumeResponse)
	if volumes.TotalItemCount == 0 {
		klog.Infof("No volumes found looking for name: %s", name)
		return nil, nil
	}

	klog.Infof("Found %v volumes with name %v", volumes.TotalItemCount, name)
	return &volumes.Items[0], nil
}

func (a *FlashBladeApi) CreateVolume(name string, sizeInBytes int64) (*ApiVolume, error) {
	loginErr := a.login()
	if loginErr != nil {
		return nil, loginErr
	}

	klog.Infof("Creating volume %s", name)
	requestPath := fmt.Sprintf("%s/file-systems?names=%s", a.apiBasePath(), name)

	body := &ApiCreateVolumeRequest{
		HardLimitEnabled: true,
		SizeInBytes:      sizeInBytes,
		Nfs:              &ApiCreateVolumeNfsConfig{V4_1_Enabled: true},
	}

	resp, err := a.restClient.R().
		SetBody(body).
		SetResult(&ApiCreateVolumeResponse{}).
		Post(requestPath)
	if err != nil {
		klog.Errorf("Creating volume failed for: %s", name)
		return nil, err
	}

	volumes := resp.Result().(*ApiCreateVolumeResponse)
	if len(volumes.Items) == 0 {
		klog.Infof("Volume response invalid after creating volume: %s", name)
		return nil, err
	}

	klog.Infof("Created volume %s successfully", name)
	return &volumes.Items[0], nil
}

func (a *FlashBladeApi) DeleteVolume(name string) error {
	loginErr := a.login()
	if loginErr != nil {
		return loginErr
	}

	klog.Infof("Deleting volume %s", name)
	requestPath := fmt.Sprintf("%s/file-systems?names=%s", a.apiBasePath(), name)

	body := &ApiDestroyVolumeRequest{
		Destroyed: true,
		Nfs:       &ApiCreateVolumeNfsConfig{V4_1_Enabled: false},
	}

	_, destroyErr := a.restClient.R().SetBody(body).Patch(requestPath)
	if destroyErr != nil {
		klog.Errorf("Destroying volume failed for: %s", name)
		return destroyErr
	}

	_, deleteErr := a.restClient.R().Delete(requestPath)
	if deleteErr != nil {
		klog.Errorf("Deleting volume failed for: %s", name)
		return deleteErr
	}

	klog.Infof("Deleted volume %s successfully", name)
	return nil
}
