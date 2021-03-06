package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/google/go-github/v39/github"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var VERSION string

var ghClient *github.Client

func init() {
	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		fmt.Println("GITHUB_TOKEN missing")
		os.Exit(1)
	}
	ghClient = github.NewClient(oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)))
}

func getAllGitHubReleases(owner, repo string) ([]*github.RepositoryRelease, error) {
	ret := make([]*github.RepositoryRelease, 0)
	opts := &github.ListOptions{Page: 1, PerPage: 100}
	for {
		releases, resp, err := ghClient.Repositories.ListReleases(context.Background(), owner, repo, opts)
		if err != nil {
			return nil, err
		}
		ret = append(ret, releases...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return ret, nil
}

func showDownloadProgressBar(name string, res *grab.Response) {
	bar := progressbar.NewOptions64(
		res.Size(),
		progressbar.OptionSetDescription(name),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetPredictTime(false),
	)
	t := time.NewTicker(100 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-t.C:
				_ = bar.Set64(res.BytesComplete())
			case <-res.Done:
				_ = bar.Finish()
				t.Stop()
				done <- struct{}{}
				return
			}
		}
	}()
	<-done
}

type RunConfig struct {
	Owner, Repo             string
	Name, License, Homepage string
	tag                     string
	publish                 bool
	NoPrefixForMainPackage  bool
}

func extractFileFromTar(rc *RunConfig, inputFile, outputFile string) error {
	compressedFile, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer compressedFile.Close()
	decompressedFile, err := gzip.NewReader(compressedFile)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(decompressedFile)
	extacted := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			if extacted {
				return nil
			}
			return fmt.Errorf("could not extract file")
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg && strings.HasPrefix(header.Name, rc.Name) {
			outFile, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return err
			}
			extacted = true
		}
	}
}

var tgzFileRe = regexp.MustCompile(`^(.*)\.(tar\.gz)|tgz$`)

func publishVersion(rc *RunConfig, tag string, assets []*github.ReleaseAsset) error {
	versionParts := strings.Split(tag, "/")
	version := strings.TrimPrefix(versionParts[len(versionParts)-1], "v")
	if !rc.publish {
		log.Println("THIS IS A DRY RUN!!! (not really publishing anything)")
	}
	log.Printf("publishing version %s", version)
	binDir := "./bin"
	log.Printf("--> creating %s", binDir)
	if err := os.RemoveAll(binDir); err != nil {
		return err
	}
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	for _, asset := range assets {
		assetName := asset.GetName()
		if assetName == "checksums.txt" {
			continue
		}
		log.Printf("--> downloading %s", assetName)
		req, err := grab.NewRequest(binDir, asset.GetBrowserDownloadURL())
		if err != nil {
			return err
		}
		res := grab.DefaultClient.Do(req)
		showDownloadProgressBar(assetName, res)
		if err := res.Err(); err != nil {
			return err
		}
		tgzFileMatch := tgzFileRe.FindAllStringSubmatch(assetName, -1)
		if len(tgzFileMatch) == 1 && len(tgzFileMatch[0]) == 3 {
			extractedAssetName := tgzFileMatch[0][1]
			log.Printf("--> extracting %s to %s", assetName, extractedAssetName)
			err = extractFileFromTar(rc, res.Filename, path.Join(binDir, extractedAssetName))
			if err != nil {
				return err
			}
			if err := os.Remove(res.Filename); err != nil {
				return err
			}
		} else if err := os.Chmod(res.Filename, 0755); err != nil {
			return err
		}
	}
	args := []string{
		"-n", rc.Name,
		"-r", version,
		"--license", rc.License,
		"--homepage", rc.Homepage,
		"--repository", fmt.Sprintf("github:%s/%s", rc.Owner, rc.Repo),
		"--package-name-prefix", "@install-binary/",
	}
	if rc.NoPrefixForMainPackage {
		args = append(args, "--no-prefix-for-main-package")
	}
	if rc.publish {
		args = append(args, "--publish")
	}

	log.Printf("--> running ./npm-binary-releaser %s", strings.Join(args, " "))
	cmd := exec.Command("./npm-binary-releaser", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runForAllReleases(rc *RunConfig) error {
	log.Println("fetching all releases...")
	allReleased, err := getAllGitHubReleases(rc.Owner, rc.Repo)
	if err != nil {
		return err
	}
	for _, release := range allReleased {
		if err := publishVersion(rc, release.GetTagName(), release.Assets); err != nil {
			return err
		}
	}
	return nil
}

func runWithConfig(rc *RunConfig) error {
	if rc.tag == "" {
		return runForAllReleases(rc)
	}
	log.Printf("fetching specific release %s", rc.tag)
	release, _, err := ghClient.Repositories.GetReleaseByTag(context.Background(), rc.Owner, rc.Repo, rc.tag)
	if err != nil {
		return err
	}
	if err := publishVersion(rc, release.GetTagName(), release.Assets); err != nil {
		return err
	}
	return nil
}

func mustReadConfig(cmd *cobra.Command) *RunConfig {
	configFile, _ := cmd.Flags().GetString("config")
	if configFile == "" {
		log.Fatalf("config flag missing")
		return nil
	}
	var rc RunConfig
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("error while reading config file %s: %v", configFile, err)
	}
	if err := json.Unmarshal(configData, &rc); err != nil {
		log.Fatalf("error while parsing config file %s: %v", configFile, err)
	}
	rc.tag, _ = cmd.Flags().GetString("tag")
	rc.publish, _ = cmd.Flags().GetBool("publish")
	return &rc
}

func run(cmd *cobra.Command, args []string) {
	if err := runWithConfig(mustReadConfig(cmd)); err != nil {
		log.Fatalf("run error: %s", err.Error())
	}
}

func main() {
	var cmd = &cobra.Command{
		Use:     "github-releases-to-npm",
		Run:     run,
		Version: VERSION,
	}

	cmd.Flags().String("tag", "", "specify release tag")
	cmd.Flags().StringP("config", "c", "", "config file")
	cmd.Flags().Bool("publish", false, "run npm publish")
	cmd.Flags().SortFlags = true

	if err := cmd.Execute(); err != nil {
		log.Fatalf("error: %s", err.Error())
	}
}
