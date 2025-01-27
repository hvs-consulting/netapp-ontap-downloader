package main

import (
	"io"
    "flag"
    "fmt"
	"net/http"
	"net/url"
	"crypto/tls"
	"encoding/json"
	"strconv"
	"mime"
	"mime/multipart"
	"os"
	"bufio"
	"math"
)

type NetAppDownloader struct {
	url string
	user string
	password string
	skip_tls bool
	volume_id string
	file_path string
	chunk_size int
}

func (n NetAppDownloader) callApi(method string, url_path string, header_accept string) (http.Header, io.ReadCloser, error) {
	transport := &http.Transport {
		TLSClientConfig: &tls.Config{InsecureSkipVerify: n.skip_tls},
	}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest(method, n.url + url_path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("Error building request: %s", err.Error())
	}

	req.SetBasicAuth(n.user, n.password)

	if header_accept != "" {
		req.Header.Set("Accept", header_accept)
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Error sending request: %s", err.Error())
	}

	return response.Header, response.Body, nil
}

func (n NetAppDownloader) GetSize() (int, error) {
	// send request
	encoded_file_path := url.QueryEscape(n.file_path)
	_, reponse_body, err := n.callApi("GET", "/api/storage/volumes/" + n.volume_id + "/files/" + encoded_file_path + "?return_metadata=true", "")
	if err != nil {
		return 0, err
	}
	defer reponse_body.Close()

	// read body
	body, err := io.ReadAll(reponse_body)
	if err != nil {
		return 0, fmt.Errorf("Error reading http body: %s", err.Error())
	}

	// parse body
	type Record struct {
		Path string
		Size int
	}
	type ResponseType struct {
		Records []Record
	}	

	var parsedResponse ResponseType
	err = json.Unmarshal(body, &parsedResponse)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse response: %s", err.Error())
	}

	// find the record (there should only be one)
	for _, record := range parsedResponse.Records {
		if record.Path == n.file_path {
			return record.Size, nil
		}
	}

	return 0, fmt.Errorf("Size not found")
}

func (n NetAppDownloader) GetChunk(bytes_start int) ([]byte, error) {
	// send request
	encoded_file_path := url.QueryEscape(n.file_path)
	response_header, response_body, err := n.callApi("GET", "/api/storage/volumes/" + n.volume_id + "/files/" + encoded_file_path + "?byte_offset=" + strconv.Itoa(bytes_start) + "&length=" + strconv.Itoa(n.chunk_size), "multipart/form-data")
	if err != nil {
		return nil, err
	}
	defer response_body.Close()

	// get multipart boundary name
	_, params, err := mime.ParseMediaType(response_header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	// parse multipart content
	reader := multipart.NewReader(response_body, params["boundary"])
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			return nil, nil
		}

		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(p)
		if err != nil {
			return nil, err
		}

		if p.Header.Get("Content-Type") == "application/octet-stream" {
			return data, nil
		}
	}

	return nil, nil
}

func main() {
    url := flag.String("url", "", "URL of NetApp ONTAP API (without /api/)")
	user := flag.String("user", "", "Username for API")
	password := flag.String("password", "", "Password for API")
	skip_tls := flag.Bool("skip-tls", false, "Skip TLS verification")
	volume_id := flag.String("volume-id", "", "ID of volume")
	file_path := flag.String("file-path", "", "Path of file to download")
	chunk_size := flag.Int("chunk-size", 1000000, "Size of one download chunk in bytes")

	output := flag.String("output", "", "Output file")

    flag.Parse()

	downloader := NetAppDownloader {
		url: *url,
		user: *user,
		password: *password,
		skip_tls: *skip_tls,
		volume_id: *volume_id,
		file_path: *file_path,
		chunk_size: *chunk_size,
	}

	// get size
	size, err := downloader.GetSize()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	// calc number of rounds
	number_of_chunks := int(math.Ceil(float64(size) / float64(*chunk_size)))

	// open file
	f, err := os.Create(*output)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	defer f.Close()

	buffer := bufio.NewWriter(f)

	// do download
	for i := 0; i < number_of_chunks; i++ {
		data, err := downloader.GetChunk(i * *chunk_size)
		if err != nil {
			fmt.Println("Error: ", err)
		}

		_, err = buffer.Write(data)
		if err != nil {
			fmt.Println("Error: ", err)
		}
	}

	// flush output file
	err = buffer.Flush()
	if err != nil {
		fmt.Println("Error: ", err)
	}
}