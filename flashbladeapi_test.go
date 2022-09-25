package main

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func setupTests(t *testing.T) *FlashBladeApi {
	err := godotenv.Load(".env.test")
	if err != nil {
		t.Log("Setup test env from .env file")
	}

	api := NewFlashBladeApi(
		os.Getenv("STORAGE_API_ADDRESS"),
		os.Getenv("STORAGE_API_TOKEN"),
		os.Getenv("SKIP_TLS_VERIFICATION") == "true",
	)
	return api
}

func TestApiLogin(t *testing.T) {
	api := setupTests(t)

	api.login()
}

func TestGetVolume(t *testing.T) {
	api := setupTests(t)

	volumeName := "ingest-data"

	volume, err := api.FindVolumeByName(volumeName)

	assert.Nil(t, err)
	if assert.NotNil(t, volume) {
		assert.Equal(t, volume.Name, volumeName)
	}
}

func TestCreateDeleteVolume(t *testing.T) {
	api := setupTests(t)

	volumeName := "test-volume"
	sizeInBytes := int64(1200000000)

	volume, createErr := api.CreateVolume(volumeName, sizeInBytes)

	assert.Nil(t, createErr)
	if assert.NotNil(t, volume) {
		assert.Equal(t, volume.Name, volumeName)
		assert.Equal(t, volume.SizeInBytes, sizeInBytes)
	}

	deleteErr := api.DeleteVolume(volumeName)
	assert.Nil(t, deleteErr)
}
