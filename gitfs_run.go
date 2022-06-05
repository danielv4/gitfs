/*
 * gitfs.go
 *
 * Copyright 2022 Daniel Vanderloo
 */
/*
 * This file is part of Cgofuse.
 *
 * It is licensed under the MIT license. The full license text can be found
 * in the License.txt file at the root of this project.
 */

package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"time"
	"bytes"
	"io/ioutil"
	"net/url"
	//"io"
	//"bufio"
	"errors"
	"encoding/base64"
)






type GithubConfig struct {

	AccessToken string
	Branch string
	Path string
}


type Github struct {

	config GithubConfig
}

type GithubUpload struct {

	Message string `json:"message"`
	Content string `json:"content"`
}


type GithubContents []struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Sha string `json:"sha"`
	Size int `json:"size"`
	URL string `json:"url"`
	HTMLURL string `json:"html_url"`
	GitURL string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type string `json:"type"`
	Links struct {
		Self string `json:"self"`
		Git string `json:"git"`
		HTML string `json:"html"`
	} `json:"_links"`
}






// WriteBuffer is a simple type that implements io.WriterAt on an in-memory buffer.
// The zero value of this type is an empty buffer ready to use.
type WriteBuffer struct {
    d []byte
    m int
}

// NewWriteBuffer creates and returns a new WriteBuffer with the given initial size and
// maximum. If maximum is <= 0 it is unlimited.
func NewWriteBuffer(size, max int) *WriteBuffer {
    if max < size && max >= 0 {
        max = size
    }
    return &WriteBuffer{make([]byte, size), max}
}

// SetMax sets the maximum capacity of the WriteBuffer. If the provided maximum is lower
// than the current capacity but greater than 0 it is set to the current capacity, if
// less than or equal to zero it is unlimited..
func (wb *WriteBuffer) SetMax(max int) {
    if max < len(wb.d) && max >= 0 {
        max = len(wb.d)
    }
    wb.m = max
}

// Bytes returns the WriteBuffer's underlying data. This value will remain valid so long
// as no other methods are called on the WriteBuffer.
func (wb *WriteBuffer) Bytes() []byte {
    return wb.d
}

// Shape returns the current WriteBuffer size and its maximum if one was provided.
func (wb *WriteBuffer) Shape() (int, int) {
    return len(wb.d), wb.m
}

func (wb *WriteBuffer) WriteAt(dat []byte, off int64) (int, error) {
    // Range/sanity checks.
    if int(off) < 0 {
        return 0, errors.New("Offset out of range (too small).")
    }
    if int(off)+len(dat) >= wb.m && wb.m > 0 {
        return 0, errors.New("Offset+data length out of range (too large).")
    }

    // Check fast path extension
    if int(off) == len(wb.d) {
        wb.d = append(wb.d, dat...)
        return len(dat), nil
    }

    // Check slower path extension
    if int(off)+len(dat) >= len(wb.d) {
        nd := make([]byte, int(off)+len(dat))
        copy(nd, wb.d)
        wb.d = nd
    }

    // Once no extension is needed just copy bytes into place.
    copy(wb.d[int(off):], dat)
    return len(dat), nil
}






func (self *Github) Post(url string, bs []byte) ([]byte, error) {
	
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(bs))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

    var client = &http.Client{Timeout: 10 * time.Second}
    r, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer r.Body.Close()
	
	arr, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }
	
	return arr, nil
}


func (self *Github) Get(url string) ([]byte, error) {
	
    req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

    var client = &http.Client{Timeout: 10 * time.Second}
    r, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer r.Body.Close()
	
	arr, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }
	
	return arr, nil
}


func (self *Github) ReadDir(dirname string) (GithubContents, error) {

	var err error
	var r GithubContents	

	bs, err := self.Get(fmt.Sprintf("https://api.github.com/repos%s/contents%s", self.config.Path, dirname))
    if err != nil {
        return r, err
    }
	
	err = json.Unmarshal(bs, &r)
	if err != nil {
		return r, err
	}	

	return r, err
}


func (self *Github) Open(fpath string) (*bytes.Reader, error) {

	var err error
	reader := new(bytes.Reader)

	req, err := http.NewRequest("GET", fmt.Sprintf("https://raw.githubusercontent.com%s/%s%s", self.config.Path, self.config.Branch, fpath), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

    var client = &http.Client{Timeout: 10 * time.Second}
    r, err := client.Do(req)
    if err != nil {
        return reader, err
    }
    defer r.Body.Close()
	
	arr, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return reader, err
    }	
	
	reader = bytes.NewReader(arr)
	return reader, err
}


func (self *Github) Create(fpath string, bs []byte) (error) {

	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write(bs)
	encoder.Close()
	
	data := GithubUpload{}
	data.Message = "client 1.0"
	data.Content = buf.String()

	bs, err := json.Marshal(data)
    if err != nil {
        return err
    }	
	
	url := fmt.Sprintf("https://api.github.com/repos%s/contents%s", self.config.Path, fpath)
	//fmt.Println(url)
	
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bs))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

    var client = &http.Client{Timeout: 10 * time.Second}
    r, err := client.Do(req)
    if err != nil {
        return err
    }
    defer r.Body.Close()
	
	arr, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return err
    }
	
	//fmt.Println(string(arr))
	return err
}


func NewClient(repo string, config GithubConfig) *Github {

	git := new(Github)
	
	u, err := url.Parse(repo)
    if err != nil {
        fmt.Println(err)
    }	
	
	config.Path = u.Path
	git.config = config
	
	return git
}



func main() {

	config := GithubConfig{}
	config.AccessToken = "AccessToken"
	config.Branch = "master"

	git := NewClient("https://github.com/username/repo", config)
	
	//git.ReadDir("/")
	
	
	
	input := []byte("foo\x00bar")
	
	err := git.Create("/eva2.h", input)
    if err != nil {
        fmt.Println(err)
    }		
	
	
	
	//file, err := git.Open("/eva.h")
    //if err != nil {
    //    fmt.Println(err)
    //}	
	//fmt.Printf("%+v\n", r)
	
	//res := make([]byte, 1024, 1024)
	//if _, err := file.ReadAt(res, int64(0)); err != nil {
	//	
	//}	
	
	//fmt.Println(string(res))
	
}