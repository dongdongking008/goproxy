package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"github.com/goproxyio/goproxy/dirhash"
	"github.com/goproxyio/goproxy/replacerule"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goproxyio/goproxy/module"
)

var cacheDir string
var listen string
var replaceManager *replacerule.RuleManager

func init() {
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
}

func main() {
	gpEnv := os.Getenv("GOPATH")
	if gpEnv == "" {
		panic("can not find $GOPATH")
	}
	gp := filepath.SplitList(gpEnv)
	rpEnv := os.Getenv("GOPROXY_REPLACERULES")
	replaceManager = replacerule.GetManager(rpEnv)
	cacheDir = filepath.Join(gp[0], "pkg", "mod", "cache", "download")
	http.Handle("/", mainHandler(http.FileServer(http.Dir(cacheDir))))
	err := http.ListenAndServe(listen, nil)
	if nil != err {
		panic(err)
	}
}

func mainHandler(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(filepath.Join(cacheDir, r.URL.Path)); err != nil {
			var suffix string
			if strings.HasSuffix(r.URL.Path, ".info") || strings.HasSuffix(r.URL.Path, ".mod") {
				suffix = ".mod"
				if strings.HasSuffix(r.URL.Path, ".info") {
					suffix = ".info"
				}
				mod := strings.Split(r.URL.Path, "/@v/")
				if len(mod) != 2 {
					ReturnServerError(w, fmt.Errorf("bad module path:%s", r.URL.Path))
					return
				}
				version := strings.TrimSuffix(mod[1], suffix)
				version, err = module.DecodeVersion(version)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				path := strings.TrimPrefix(mod[0], "/")
				path, err := module.DecodePath(path)
				if err != nil {
					ReturnServerError(w, err)
					return
				}
				// ignore the error, incorrect tag may be given
				// forward to inner.ServeHTTP
				goGet(path, version, suffix, w, r)
			}
			if strings.HasSuffix(r.URL.Path, "/@v/list") {
				w.WriteHeader(200)
				w.Write([]byte(""))
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
}

func goGet(opath, version, suffix string, w http.ResponseWriter, r *http.Request) error {
	path := replaceManager.Replace(opath)
	cmd := exec.Command("go", "get", "-d", path+"@"+version)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stdout, "goproxy: download %s stdout: %s stderr: %s\n", path, string(bytesOut), string(bytesErr))
		return err
	}
	out := fmt.Sprintf("%s", bytesErr)

	if opath != path {
		err := replaceMod(opath, path)
		if err != nil {
			fmt.Printf("goproxy: copy %s to %s error: %s\n", path, opath, err)
			return err
		}
	}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			continue
		}
		if f[1] == "downloading" && f[2] == path && f[3] != version {
			h := r.Host
			mod := strings.Split(r.URL.Path, "/@v/")
			p := fmt.Sprintf("%s/@v/%s%s", mod[0], f[3], suffix)
			scheme := "http:"
			if r.TLS != nil {
				scheme = "https:"
			}
			url := fmt.Sprintf("%s//%s/%s", scheme, h, p)
			http.Redirect(w, r, url, 302)
		}
	}
	return nil
}

func replaceMod(opath string, path string) error {
	oDir := filepath.Join(cacheDir, opath)
	dir := filepath.Join(cacheDir, path)
	if _, err := os.Stat(oDir); err != nil && os.IsNotExist(err) {
		os.MkdirAll(oDir, os.ModeDir)
	}

	cpCMD := fmt.Sprintf("cp -rfu %s/* %s", dir, oDir)
	cmd := exec.Command("/bin/sh", "-c", cpCMD)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stdout, "goproxy: cp %s to %s stdout: %s stderr: %s\n", path, opath, string(bytesOut), string(bytesErr))
		return err
	}

	zipFiles, err := filepath.Glob(oDir + "/@v/*.zip")
	if err != nil {
		return err
	}
	for _, f := range zipFiles {
		err := replaceZip(f, path, opath)
		if err != nil {
			return err
		}
	}
	return nil
}

func replaceZip(zipFile string, path string, opath string) error {
	tempZipFile := zipFile + ".tmp"
	err := copyZip(zipFile, tempZipFile, path, opath)
	if err != nil {
		return err
	}

	cmd := exec.Command("mv", tempZipFile, zipFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}

	bytesOut, err := ioutil.ReadAll(stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("goproxy: mv %s %s stdout: %s stderr: %s\n", tempZipFile, zipFile, string(bytesOut), string(bytesErr))
		return err
	}

	ziphash, err := dirhash.HashZip(zipFile, dirhash.Hash1)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(zipFile+"hash", []byte(ziphash), os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}

func copyZip(zipFile string, tempZipFile string, path string, opath string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	fileWrite, err := os.Create(tempZipFile)
	if err != nil {
		return err
	}
	defer fileWrite.Close()

	zipWriter := zip.NewWriter(fileWrite)
	for _, file := range zipReader.File {

		rc, err := file.Open()
		if err != nil {
			return err
		}

		zipHeader := file.FileHeader
		zipHeader.Name = strings.Replace(zipHeader.Name, path, opath, 1)

		f, err := zipWriter.CreateHeader(&zipHeader)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
		rc.Close()
	}

	err = zipWriter.Close()
	if err != nil {
		return err
	}
	return nil
}
