/**
 * go-mapbox Base Module
 * Provides a common base for API modules
 * See https://www.mapbox.com/api-documentation/ for API information
 *
 * https://github.com/ryankurte/go-mapbox
 * Copyright 2017 Ryan Kurte
 */

package base

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

const (
	// BaseURL Mapbox API base URL
	BaseURL = "https://api.mapbox.com"

	statusRateLimitExceeded = 429
)

// Base Mapbox API base
type Base struct {
	token string
	debug bool
}

// NewBase Create a new API base instance
func NewBase(token string) (*Base, error) {
	if token == "" {
		return nil, errors.New("Mapbox API token not found")
	}

	b := &Base{}

	b.token = token

	return b, nil
}

// SetDebug enables debug output for API calls
func (b *Base) SetDebug(debug bool) {
	b.debug = true
}

//MapboxAPIMessage simple holder for responses from MapBox
type MapboxAPIMessage struct {
	Message string
}

//SimpleGET for the status check
func (b *Base) SimpleGET(url string) ([]byte, error) {
	url = fmt.Sprintf("%s/%s?access_token=%s", BaseURL, url, b.token)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, _ := ioutil.ReadAll(response.Body)
	return body, nil

}

//PostRequest sends a simple json/application post request
func (b *Base) PostRequest(postURL string, data []byte) ([]byte, error) {
	postURL = fmt.Sprintf("%s/%s?access_token=%s", BaseURL, postURL, b.token)

	request, err := http.NewRequest(http.MethodPost, postURL, bytes.NewBuffer(data))

	if data != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	return body, nil

}

//PostUploadFileRequest sends multipart/form-data POST request to the mapbox api
func (b *Base) PostUploadFileRequest(postURL string, file string, filetype string) ([]byte, error) {

	geoJSON, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer geoJSON.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(filetype, filepath.Base(geoJSON.Name()))
	if err != nil {
		return nil, err
	}
	io.Copy(part, geoJSON)
	writer.Close()

	postURL = fmt.Sprintf("%s/%s/?access_token=%s", BaseURL, postURL, b.token)
	request, err := http.NewRequest(http.MethodPost, postURL, body)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}

	response, err := client.Do(request)
	resBody, err := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		apiMessage := MapboxAPIMessage{}
		messageErr := json.Unmarshal(resBody, &apiMessage)
		if messageErr == nil {
			return nil, fmt.Errorf("api error: %s", apiMessage.Message)
		}
		return nil, fmt.Errorf("Bad Request (400) - no message")
	}
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	fmt.Println(string(resBody))

	return resBody, nil
}

// QueryRequest make a get with the provided query string and return the response if successful
func (b *Base) QueryRequest(query string, v *url.Values) (*http.Response, error) {
	// Add token to args
	v.Set("access_token", b.token)

	// Generate URL
	url := fmt.Sprintf("%s/%s", BaseURL, query)

	if b.debug {
		fmt.Printf("URL: %s\n", url)
	}

	// Create request object
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = v.Encode()

	// Create client instance
	client := &http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if b.debug {
		data, _ := httputil.DumpRequest(request, true)
		fmt.Printf("Request: %s", string(data))
		data, _ = httputil.DumpResponse(resp, false)
		fmt.Printf("Response: %s", string(data))
	}

	if resp.StatusCode == statusRateLimitExceeded {
		return nil, ErrorAPILimitExceeded
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrorAPIUnauthorized
	}

	return resp, nil
}

// QueryBase Query the mapbox API and fill the provided instance with the returned JSON
// TODO: Rename this
func (b *Base) QueryBase(query string, v *url.Values, inst interface{}) error {
	// Make request
	resp, err := b.QueryRequest(query, v)
	if err != nil && (resp == nil || resp.StatusCode != http.StatusBadRequest) {
		return err
	}
	defer resp.Body.Close()

	// Read body into buffer
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Handle bad requests with messages
	if resp.StatusCode == http.StatusBadRequest {
		apiMessage := MapboxAPIMessage{}
		messageErr := json.Unmarshal(body, &apiMessage)
		if messageErr == nil {
			return fmt.Errorf("api error: %s", apiMessage.Message)
		}
		return fmt.Errorf("Bad Request (400) - no message")
	}

	// Attempt to decode body into inst type
	err = json.Unmarshal(body, &inst)
	if err != nil {
		return err
	}

	return nil
}

// Query the mapbox API
// TODO: Depreciate this
func (b *Base) Query(api, version, mode, query string, v *url.Values, inst interface{}) error {

	// Generate URL
	queryString := fmt.Sprintf("%s/%s/%s/%s", api, version, mode, query)

	return b.QueryBase(queryString, v, inst)
}
