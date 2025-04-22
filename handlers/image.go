package handlers

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
	"go_/structs"
	"math/rand"
	"strconv"
	"time"
)

var ErrResponseBodyEmpty = errors.New("response body is empty or not a map")

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
	result, err := fetchPixivIllustDataFromPixiv(strconv.Itoa(pid), "http://127.0.0.1:7890")
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
	name := getIllustInformationFromPixivIllust(result)
	urls := getUrlsFromPixivIllust(result)
	author := structs.Author{
		Name: getUserNameFromPixivIllust(result),
		UID:  getUserIdFromPixivIllust(result),
	}
	author, err = database.GetOrCreateAuthor(author)

	_, err = database.CreateImage(pid, name, author.ID, urls)
	tags := getTagsFromPixivIllust(result)
	for _, tag := range tags {
		tid, err := database.GetOrCreateTagIdByName(tag)
		if err != nil {
			return err
		}
		err = database.InsertImageTag(pid, tid)
	}
	return nil
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
