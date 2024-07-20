package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
	"go_/structs"
	"go_/utils"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var ErrResponseBodyEmpty = errors.New("response body is empty or not a map")

func getInformationFromPid(pid int) (map[string]interface{}, error) {
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	url := "https://www.pixiv.net/ajax/illust/" + strconv.Itoa(pid)
	method := "GET"
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
	}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", utils.Cookies)
	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0")
	req.Header.Add("x-requested-with", "XMLHttpRequest")
	req.Header.Add("referer", "https://www.pixiv.net/artworks/100000000")
	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("获取Tag时发生错误")
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	//fmt.Println(res.Body)
	if err != nil {
		return nil, err
	}
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Error().Err(err).Msg("获取Tag时发生错误")
		return nil, err
	}
	bodyContent, ok := response["body"].(map[string]interface{})
	if !ok || bodyContent == nil {
		log.Error().Msg("response body is empty or not a map")
		// 在这里可以返回一个特殊的错误或 nil, nil，取决于你的需求
		return nil, ErrResponseBodyEmpty
	}

	return bodyContent, nil
}
func getTagsFromResult(result map[string]interface{}) []string {
	var tagNames []string
	tags, _ := result["tags"].(map[string]interface{})
	tagList, _ := tags["tags"].([]interface{})
	for _, tagItem := range tagList {
		tagMap, _ := tagItem.(map[string]interface{})
		tagName, _ := tagMap["tag"].(string)
		tagNames = append(tagNames, tagName)
	}
	return tagNames
}
func getIllustInformationFromResult(result map[string]interface{}) string {
	var illustTitle string
	illustTitle = result["illustTitle"].(string)
	return illustTitle
}
func getUserIdFromResult(result map[string]interface{}) string {
	var userId string
	userId = result["userId"].(string)
	return userId
}
func getUserNameFromResult(result map[string]interface{}) string {
	var userName string
	userName = result["userName"].(string)
	return userName
}
func pixivHandler(pid int, path string, fileType string) error {
	rand.Seed(time.Now().UnixNano())
	min := 0.1
	max := 1.0
	randomDuration := time.Duration(min*float64(time.Second) + rand.Float64()*(max-min)*float64(time.Second))
	time.Sleep(randomDuration)
	exist, err := database.CheckPidExists(pid)
	if exist == true {
		return nil
	}
	result, err := getInformationFromPid(pid)
	if err != nil {
		//_, err = database.CreateImage(pid, "", path, 0)
		if errors.Is(err, ErrResponseBodyEmpty) {
			//tid, err := database.GetOrCreateTagIdByName("由于作者删除该作品无法获得tag")
			//if err != nil {
			//	return err
			//}
			//err = database.InsertImageTag(pid, tid)
		}
		return err
	}
	name := getIllustInformationFromResult(result)
	author := structs.Author{
		Name: getUserNameFromResult(result),
		UID:  getUserIdFromResult(result),
	}
	author, err = database.GetOrCreateAuthor(author)
	_, err = database.CreateImage(pid, name, path, author.ID, fileType)
	tags := getTagsFromResult(result)
	for _, tag := range tags {
		tid, err := database.GetOrCreateTagIdByName(tag)
		if err != nil {
			return err
		}
		err = database.InsertImageTag(pid, tid)
	}
	return nil
}

func getImageByPid(ctx *fiber.Ctx) error {
	var image structs.Image
	var err error
	pidStr := ctx.Params("pid")
	pid, err := strconv.Atoi(pidStr)
	log.Log().Msg("pid=" + pidStr)
	if err != nil {
		log.Error().Err(err)
		return sendCommonResponse(ctx, 500, "转换id有误", nil)
	}
	image, err = database.GetImageById(pid)
	if err != nil {
		log.Error().Err(err)
		return sendCommonResponse(ctx, 500, "查询图片出现错误", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"pid":    image.PID,
		"author": image.Author,
		"tags":   image.Tags,
		"path":   image.Path,
	})
}

type SearchRequest struct {
	Tags      []string `json:"tags"`
	Page      int      `json:"page"`
	PageSize  int      `json:"size"`
	SortBy    string   `json:"sort_by"`
	SortOrder string   `json:"sort_order"`
	Author    string   `json:"author"`
}

func searchImages(ctx *fiber.Ctx) error {
	var req SearchRequest

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse JSON",
		})
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	if req.SortBy == "pid" || req.SortBy == "" {
		req.SortBy = "i.pid"
	}

	if req.SortOrder == "" {
		req.SortOrder = "DESC"
	}
	if req.Tags == nil || len(req.Tags) == 0 {
		var count int
		images, count, err := database.GetImagesWithPagination(req.Page, req.PageSize, req.Author, req.SortBy, req.SortOrder)
		if err != nil {
			log.Error().Err(err)
			return sendCommonResponse(ctx, 500, "查询图片出现错误", nil)
		}
		return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
			"images": images,
			"total":  count,
		})
	} else {
		var count int
		images, count, err := database.SearchImages(req.Tags, req.Page, req.PageSize, req.Author, req.SortBy, req.SortOrder)
		if err != nil {
			log.Error().Err(err)
			return sendCommonResponse(ctx, 500, "查询图片出现错误", nil)
		}
		return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
			"images": images,
			"total":  count,
		})
	}
}
