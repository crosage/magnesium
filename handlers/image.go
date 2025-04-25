package handlers

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
	"go_/structs"
	"math/rand"
	"strconv"
	"time"
)

var ErrResponseBodyEmpty = errors.New("response body is empty or not a map")

func pixivHandler(pid int) error {
	rand.Seed(time.Now().UnixNano())
	min := 0.1
	max := 5.0
	randomDuration := time.Duration(min*float64(time.Second) + rand.Float64()*(max-min)*float64(time.Second))
	time.Sleep(randomDuration)

	exists, err := database.CheckPidExists(pid)
	if err != nil {
		return fmt.Errorf("error checking pid %d existence: %w", pid, err)
	}
	result, err := fetchPixivIllustDataFromPixiv(strconv.Itoa(pid), "http://127.0.0.1:7890")
	if err != nil {
		if errors.Is(err, ErrResponseBodyEmpty) {
		}
		return fmt.Errorf("error fetching pixiv data for pid %d: %w", pid, err)
	}
	name := getIllustInformationFromPixivIllust(result)
	urls := getUrlsFromPixivIllust(result)
	bookmarkCount := getBookmarkCountFromPixivIllust(result)
	isBookmarked := getBookmarkFromPixivIllust(result)
	authorInfo := structs.Author{
		Name: getUserNameFromPixivIllust(result),
		UID:  getUserIdFromPixivIllust(result),
	}

	author, err := database.GetOrCreateAuthor(authorInfo)

	if err != nil {
		return fmt.Errorf("error getting or creating author for pid %d: %w", pid, err)
	}

	if exists {
		err = database.UpdateImage(pid, name, author.ID, bookmarkCount, isBookmarked, urls)
		if err != nil {
			return fmt.Errorf("error updating image record for pid %d: %w", pid, err)
		}
		err = database.DeleteImageTags(pid)
		if err != nil {
			return fmt.Errorf("error clearing old tags for pid %d: %w", pid, err)
		}
	} else {
		_, err = database.CreateImage(pid, name, author.ID, bookmarkCount, isBookmarked, urls)
		if err != nil {
			return fmt.Errorf("error creating image record for pid %d: %w", pid, err)
		}
	}

	tags := getTagsFromPixivIllust(result)
	for _, tagName := range tags {
		tid, err := database.GetOrCreateTagIdByName(tagName)
		if err != nil {
			return fmt.Errorf("error getting or creating tag id for tag '%s' (pid %d): %w", tagName, pid, err)
		}
		err = database.InsertImageTag(pid, tid)
		if err != nil {
			return fmt.Errorf("error inserting image-tag link for pid %d, tag id %d: %w", pid, tid, err)
		}
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
		rawBody := ctx.Body() // 读取原始 Body
		log.Printf("Received Raw Body: %s", string(rawBody))
		log.Error().Err(err).Msg("解析报文出现错误")
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
