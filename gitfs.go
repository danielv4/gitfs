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
	"os"
	"fmt"
	
	"github.com/winfsp/cgofuse/fuse"
	
	//"io"
	//"path"
	
	"net/http"
	"encoding/json"
	"bytes"
	"io/ioutil"
	"net/url"
	"errors"
	
	"encoding/base64"
)



// set CPATH=C:\Program Files (x86)\WinFsp\inc\fuse


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
	Sha string `json:"sha"`
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


type GithubStat struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	GitURL      string `json:"git_url"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Links       struct {
		Self string `json:"self"`
		Git  string `json:"git"`
		HTML string `json:"html"`
	} `json:"_links"`
}




func (self *Github) Get(url string) ([]byte, error) {
	
    req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

    var client = &http.Client{}
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
	
	url := fmt.Sprintf("https://api.github.com/repos%s/contents%s?branch=%s", self.config.Path, fpath, self.config.Branch)

	fmt.Println(url)

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %s", self.config.AccessToken))
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	

    var client = &http.Client{}
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
	
	stat, err := self.Stat(fpath)
	if stat.Sha != "" {
		data.Sha = stat.Sha
	}
	
	data.Content = buf.String()

	jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }	
	
	url := fmt.Sprintf("https://api.github.com/repos%s/contents%s", self.config.Path, fpath)
	//fmt.Println(url)
	
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

    var client = &http.Client{}
    r, err := client.Do(req)
    if err != nil {
        return err
    }
    defer r.Body.Close()
	
	_, err = ioutil.ReadAll(r.Body)
    if err != nil {
        return err
    }
	
	//fmt.Println(string(arr))
	return err
}


func (self *Github) Stat(fpath string) (GithubStat, error) {

	var err error
	var stat GithubStat

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos%s/contents%s?branch=%s", self.config.Path, fpath, self.config.Branch), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))
	req.Header.Set("Content-Type", "application/json")

    var client = &http.Client{}
    r, err := client.Do(req)
    if err != nil {
        return stat, err
    }
    defer r.Body.Close()
	
	arr, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return stat, err
    }	

	err = json.Unmarshal(arr, &stat)
	if err != nil {
		return stat, err
	}

	return stat, err
}


func (self *Github) Remove(fpath string) (error) {

	var err error
	
	
	data := GithubUpload{}
	data.Message = "client 1.0"
	stat, err := self.Stat(fpath)
	//fmt.Printf("Sha => %s\n", stat.Sha)
	if stat.Sha != "" {
		data.Sha = stat.Sha
	}

	jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }	

	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.github.com/repos%s/contents%s?branch=%s", self.config.Path, fpath, self.config.Branch), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", self.config.AccessToken))

    var client = &http.Client{}
    r, err := client.Do(req)
    if err != nil {
        return err
    }
    defer r.Body.Close()
	
	_, err = ioutil.ReadAll(r.Body)
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
















// Cache
type Node struct {
	
	Path    string
	IsDir   bool
	Size    int
	fp      *bytes.Reader
	mknod   *WriteBuffer
}


type Gitfs struct {
	fuse.FileSystemBase
	client *Github
	nodes map[string]*Node
}







func (self *Gitfs) Mknod(path string, mode uint32, dev uint64) (errc int) {

	// pre_write
	// create file 
	// then open
	//fmt.Printf("Mknod => %s\n", path)
	

	fp := NewWriteBuffer(0, 9999999)
	
	node := new(Node)
	node.IsDir = false
	node.Size = 0
	node.Path = path	
	node.mknod = fp
	self.nodes[path] = node

	return
}


func (self *Gitfs) Write(path string, buff []byte, ofst int64, fh uint64) (n int) {

	//fmt.Printf("Write() %s\n", path)
	//fmt.Printf("Write(?) %d\n", len(buff))

	if node, found := self.nodes[path]; found {
	
		n, err := node.mknod.WriteAt(buff, ofst)
		if nil != err {
			//n = fuseErrc(err)
			return 0
		}
		//fmt.Printf("Written => %d \n", n)
		
		self.nodes[path].Size += n

		return n		
	} else {
		n = -fuse.EIO
		return 0 
	}

	return 0
}


func (self *Gitfs) Open(path string, flags int) (errc int, fh uint64) {

	//fmt.Printf("Open() %s\n", path)

	if _, found := self.nodes[path]; found {
	
		//OpenFile(path string, f int) (*File, error)
		return 0, 0
		
	} else {
		return -fuse.ENOENT, ^uint64(0)
	}
}


func (self *Gitfs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {

	//fmt.Printf("Read() %s\n", path)

	if node, found := self.nodes[path]; found {
	
		if node.fp == nil {
			fp, err := self.client.Open(path)
			if err != nil {
				fmt.Println(err)
			}
			self.nodes[path].fp = fp
		}		
	
	
		n, err := node.fp.ReadAt(buff, ofst)
		if nil != err {
			//n = fuseErrc(err)
			return 0
		}

		return n		
	} else {
		n = -fuse.EIO
		return 0 
	}

	return 0
}


func (self *Gitfs) Mkdir(path string, mode uint32) (errc int) {
	// pre_write
	// create file 
	// then open
	//fmt.Printf("Mkdir => %s\n", path)
	
	//err := self.client.MkdirAll(path)
	//if err != nil {
	//	return
	//}
	
	node := new(Node)
	node.IsDir = true
	node.Size = 0
	node.Path = path	
	self.nodes[path] = node

	return
}


func (self *Gitfs) Unlink(path string) (errc int) {
	
	err := self.client.Remove(path)
	if err != nil {
		fmt.Println(err)
	}
	return 0
}


func (self *Gitfs) Rmdir(path string) (errc int) {
	
	return 0
}


func (self *Gitfs) Opendir(path string) (errc int, fh uint64) {
	//fmt.Printf("Opendir() %s\n", path)
	return 0, 0
}


func (self *Gitfs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {

	//fmt.Printf("Getattr() %s\n", path)
	//fmt.Printf("%+v\n", self.nodes)
	
	if path == "/" {
		stat.Mode = fuse.S_IFDIR | 0777
		return 0	
	} else if node, found := self.nodes[path]; found {
	
		if node.IsDir == true {
			stat.Mode = fuse.S_IFDIR | 0777
		} else {
			stat.Mode = fuse.S_IFREG | 0777
			stat.Size = int64(node.Size)	
		}

		return 0		
	} else {
		return -fuse.ENOENT
	}
}


func (self *Gitfs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	
	
	fill(".", nil, 0)
	fill("..", nil, 0)
	
	
	entries, err := self.client.ReadDir(path)
	if err != nil {
		fmt.Println(err)
	} else {
	
		//self.updateInodes(path, entries)
	
		for _, entry := range entries {
		
			fill(entry.Name, nil, 0)
			
			// add node to Cache for Getattr()
			node := new(Node)
			
			if entry.Type == "dir" {
				node.IsDir = true
			} else if entry.Type == "file" {
				node.IsDir = false
			}
			
			node.Size = int(entry.Size)
			if path == "/" {
				node.Path = path + entry.Name
			} else {
				node.Path = path + "/" + entry.Name
			}
			
			//fmt.Printf("%+v\n", node)
			
			self.nodes[node.Path] = node
		}
	}
	

	return 0
}


func (self *Gitfs) Release(path string, fh uint64) (errc int) {
	
	//fmt.Printf("Release() %s\n", path)
	
	if node, found := self.nodes[path]; found {

		if len(node.mknod.d) > 0 {
		
			//fmt.Printf("[+] github Creating file \n")
			err := self.client.Create(path, node.mknod.d)
			if err != nil {
				fmt.Println(err)
			} 
		}	
	}	
	
	return 0
}


func (self *Gitfs) Statfs(path string, stat *fuse.Statfs_t) (err int) {
	
	//fmt.Printf("STAT FS!!! %s\n", path)
	stat.Bsize = 4096
	// f_frsize
	stat.Frsize = 4096

	// 8 EB - 1
	vtotal := (8 << 50) / stat.Frsize * 1024 - 1
	vavail := (2 << 50) / stat.Frsize * 1024
	vfree  := (1 << 50) / stat.Frsize * 1024
	//used := total - free

	// f_blocks
	stat.Blocks = vtotal
	stat.Bfree  = vfree
	stat.Bavail = vavail

	stat.Files  = 2240224
	stat.Ffree  = 1927486
	stat.Favail = 9900000

	stat.Namemax = 255
	return 0
}


func main() {

	gitfs := &Gitfs{}
	
	config := GithubConfig{}
	config.AccessToken = "AccessToken"
	config.Branch = "main"
	git := NewClient("https://github.com/username/repo2", config)
	
	
	// init
	gitfs.client = git
	gitfs.nodes = make(map[string]*Node)
	
	
	host := fuse.NewFileSystemHost(gitfs)
	host.SetCapReaddirPlus(true)
	host.Mount("", append([]string{
		"-o", "ExactFileSystemName=NTFS",
		"-o", fmt.Sprintf("volname=%s", "Nice"),
	}, os.Args[1:]...))	
}
