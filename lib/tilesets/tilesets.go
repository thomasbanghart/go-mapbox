package tilesets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ryankurte/go-mapbox/lib/base"
)

const (
	apiName    = "tilesets"
	apiVersion = "v1"
)

//Tileset struct holds base query
type Tileset struct {
	base      *base.Base
	username  string
	tilesetID string
}

//UploadResponse  a simple struct for response for upload endpoint
type UploadResponse struct {
	FileSize   int    `json:"file_size"`
	Files      int    `json:"files"`
	SourceSize int    `json:"source_size"`
	ID         string `json:"id"`
}

//PublishResponse a simple struct for response for publish endpoint
type PublishResponse struct {
	Message string `json:"message"`
	JobID   string `json:"jobId"`
}

type StatusResponse struct {
	ID        string `json:"id"`
	LatestJob string `json:"latest_job"`
	Status    string `json:"status"`
}

//NewTileset constructs a new tileset
func NewTileset(base *base.Base) *Tileset {
	return &Tileset{base, "", ""}
}

//SetTileset helper function to set values for calls
func (t *Tileset) SetTileset(username string, tilesetID string) {
	t.username = username
	t.tilesetID = tilesetID
}

func (t *Tileset) postURL() string {
	return fmt.Sprintf(
		"%s/%s/%s",
		apiName,
		apiVersion,
		fmt.Sprintf("%s.%s", t.username, t.tilesetID),
	)
}

//UploadGeoJSON writes a line-delimiated geoJSON file to a mapbox account
func (t *Tileset) UploadGeoJSON(pathToFile string) (*UploadResponse, error) {
	postURL := fmt.Sprintf("%s/%s/sources/%s/%s", apiName, apiVersion, t.username, t.tilesetID)
	uploadResponse := &UploadResponse{}
	res, err := t.base.PostUploadFileRequest(postURL, pathToFile, "file")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(res, uploadResponse)
	if err != nil {
		log.Fatal(err)
	}
	return uploadResponse, nil

}

//CreateTileset creates tileset based on tileset-recipe.json file
func (t *Tileset) CreateTileset(pathToFile string) (*base.MapboxAPIMessage, error) {
	maboxAPIMessage := &base.MapboxAPIMessage{}
	recipeJSON, err := os.Open(pathToFile)
	if err != nil {
		return nil, err
	}
	defer recipeJSON.Close()

	data, _ := ioutil.ReadAll(recipeJSON)
	res, err := t.base.PostRequest(t.postURL(), data)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(res, &maboxAPIMessage)

	return maboxAPIMessage, nil

}

//PublishTileset publishes a tileset after it has been created
func (t *Tileset) PublishTileset() (*PublishResponse, error) {
	publishResponse := &PublishResponse{}
	res, err := t.base.PostRequest(t.postURL()+"/publish", nil)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(res))
	json.Unmarshal(res, &publishResponse)
	return publishResponse, nil
}

//CheckJobStatus shows the stuats of the most recent job
func (t *Tileset) CheckJobStatus() error {
	fmt.Println("Awaiting job completion. This may take some time...")
	for {
		statusResponse := &StatusResponse{}
		res, err := t.base.SimpleGET(t.postURL() + "/status")
		if err != nil {
			return err
		}
		json.Unmarshal(res, statusResponse)
		if statusResponse.Status == "failed" {
			fmt.Println("Job failed")
			return nil
		}
		if statusResponse.Status == "success" {
			fmt.Println("Job complete")
			return nil
		}
		fmt.Println(statusResponse.Status)
		time.Sleep(5 * time.Second)

	}

}
