package charthelper

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yunionio/log"

	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/types/helm"
)

const defaultTimeout time.Duration = 30 * time.Second

// DownloadAndExtractChartTarball the chart tart file linked by metadata.Urls and store
// the wanted files (i.e. README.md) under chartDataDir
func DownloadAndExtractChaartTarball(chart *helm.ChartPackage, repoUrl string) (err error) {
	if err := ensureChartDataDir(chart); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			cleanChartDataDir(chart)
		}
	}()

	if !tarballExists(chart) {
		if err := downloadTarball(chart, repoUrl); err != nil {
			return err
		}
	}

	if err := extractFilesFromTarball(chart); err != nil {
		return err
	}

	return nil
}

func tarballExists(chart *helm.ChartPackage) bool {
	_, err := os.Stat(TarballPath(chart))
	return err == nil
}

func downloadTarball(chart *helm.ChartPackage, repoUrl string) error {
	source := chart.Urls[0]
	if _, err := url.ParseRequestURI(source); err != nil {
		// If the chart URL is not absolute, join the repo URL. It's fine if the
		// URL we build here is invalid as we can caatch this error when actually
		// making the request
		u, _ := url.Parse(repoUrl)
		u.Path = path.Join(u.Path, source)
		source = u.String()
	}

	destination := TarballPath(chart)

	// Create output
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	log.Infof("Downloading metadata, source: %q, dest: %q", source, destination)
	// Download tarball
	c := &http.Client{
		Timeout: defaultTimeout,
	}
	req, err := http.NewRequest("GET", source, nil)
	req.Header.Set("User-Agent", "yunion-kube")
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		log.Errorf("Error downloading %s, %d", source, resp.StatusCode)
	}

	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

var filesToKeep = []string{"README.md"}

func extractFilesFromTarball(chart *helm.ChartPackage) error {
	tarballPath := TarballPath(chart)
	tarballExpandedPath, err := ioutil.TempDir(os.TempDir(), "chart")
	if err != nil {
		return err
	}

	// Extract
	if _, err := os.Stat(tarballPath); err != nil {
		return fmt.Errorf("Can't find file to extract %s", tarballPath)
	}

	if err := untar(tarballPath, tarballExpandedPath); err != nil {
		return err
	}

	// Save specific files defined by filesToKeep
	// Include /[chartName] namespace
	chartPath := filepath.Join(tarballExpandedPath, chart.Name)
	for _, fileName := range filesToKeep {
		src := filepath.Join(chartPath, fileName)
		dest := filepath.Join(chartDataDir(chart), fileName)

		log.Infof("Storing in cache path: %q", dest)

		if err := copyFile(dest, src); err != nil {
			return err
		}
	}
	return nil
}

func ensureChartDataDir(chart *helm.ChartPackage) error {
	dir := chartDataDir(chart)
	if _, err := os.Stat(dir); err != nil {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Cound not create dir %s: %s", dir, err)
		}
	}
	return nil
}

func TarballPath(chart *helm.ChartPackage) string {
	return filepath.Join(chartDataDir(chart), "chart.tgz")
}

func cleanChartDataDir(chart *helm.ChartPackage) error {
	return os.RemoveAll(chartDataDir(chart))
}

func chartDataDir(chart *helm.ChartPackage) string {
	return filepath.Join(DataDirBase(), chart.Repo, chart.Name, chart.Version)
}

func DataDirBase() string {
	return filepath.Join(options.Options.HelmDataDir, "repo-data")
}

func untar(tarball, dir string) error {
	r, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer r.Close()

	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// split header name and create missing directories
		d, _ := filepath.Split(header.Name)
		fullDir := filepath.Join(dir, d)
		_, err = os.Stat(fullDir)
		if err != nil && d != "" {
			if err = os.MkdirAll(fullDir, 0700); err != nil {
				return err
			}
		}

		path := filepath.Clean(filepath.Join(dir, header.Name))
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tr)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyFile(dst, src string) error {
	i, err := os.Open(src)
	if err != nil {
		return err
	}
	defer i.Close()

	o, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer o.Close()

	_, err = io.Copy(o, i)

	return err
}

func ReadFromCache(chart *helm.ChartPackage, fileName string) (string, error) {
	fileToLoad := filepath.Join(chartDataDir(chart), fileName)
	dat, err := ioutil.ReadFile(fileToLoad)
	if err != nil {
		return "", fmt.Errorf("Can't find local file %s", fileToLoad)
	}
	return string(dat), nil
}

func ChartDataExists(chart *helm.ChartPackage) (bool, error) {
	_, err := os.Stat(chartDataDir(chart))
	if err != nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ReadmeStaticUrl returns the static path for the README.md file
func ReadmeStaticUrl(chart *helm.ChartPackage, prefix string) string {
	path := filepath.Join(chartDataDir(chart), "README.md")
	return staticUrl(path, prefix)
}

func staticUrl(path, prefix string) string {
	return strings.Replace(path, DataDirBase(), prefix, 1)
}
