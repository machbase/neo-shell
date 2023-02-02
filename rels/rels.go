package rels

import (
	"fmt"
	"io"
	"net/http"

	"github.com/tidwall/gjson"
)

func FetchGithubReleases(org string, repo string) error {
	path := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", org, repo)
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("http response %s", rsp.Status)
	}

	buff, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}
	payload := string(buff)

	releases := make([]Release, 0)
	rs := gjson.Get(payload, "@this").Array()
	for i, r := range rs {
		rel := Release{}
		rel.TagName = r.Get("tag_name").Str
		rel.PublishedAt = r.Get("published_at").Str
		rel.Name = r.Get("name").Str
		rel.Draft = r.Get("draft").Bool()
		rel.PreRelease = r.Get("prerelease").Bool()
		rel.Assets = make([]Asset, 0)
		releases = append(releases, rel)

		fmt.Println(i, rel.TagName, rel.PublishedAt, rel.Name)
		fmt.Println("   draft:", rel.Draft, " prerelease:", rel.PreRelease)

		assets := r.Get("assets").Array()
		for _, entry := range assets {
			asset := Asset{}
			asset.Name = entry.Get("name").Str
			asset.DownloadCount = entry.Get("download_count").Int()
			asset.Size = entry.Get("size").Int()
			asset.ContentType = entry.Get("content_type").Str
			asset.State = entry.Get("state").Str
			asset.CreatedAt = entry.Get("created_at").Str
			asset.UpdatedAt = entry.Get("updated_at").Str
			asset.DownloadUrl = entry.Get("browser_download_url").Str
			rel.Assets = append(rel.Assets, asset)

			fmt.Println("    name  ", asset.Name)
			fmt.Println("       downloads", asset.DownloadCount)
			fmt.Println("            size", asset.Size)
			fmt.Println("    content_type", asset.ContentType)
			fmt.Println("           state", asset.State)
			fmt.Println("      created_at", asset.CreatedAt)
			fmt.Println("      updated_at", asset.UpdatedAt)
			fmt.Println("             url", asset.DownloadUrl)
		}
	}
	return nil
}

type Release struct {
	Name        string
	TagName     string
	PublishedAt string
	Draft       bool
	PreRelease  bool
	Assets      []Asset
}

type Asset struct {
	Name          string
	DownloadCount int64
	Size          int64
	ContentType   string
	State         string
	CreatedAt     string
	UpdatedAt     string
	DownloadUrl   string
}
