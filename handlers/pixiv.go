package handlers

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"go_/database"
	"go_/structs"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPixivRequestFailed  = errors.New("failed to execute request to Pixiv API")
	ErrPixivBadStatus      = errors.New("pixiv API returned non-OK status")
	ErrPixivReadBodyFailed = errors.New("failed to read Pixiv response body")
	ErrPixivParseFailed    = errors.New("failed to parse JSON data from Pixiv")
	ErrInternalSetupFailed = errors.New("internal setup failed (request creation/headers)")
)

type BookmarkPayload struct {
	IllustID string   `json:"illust_id"`
	Restrict int      `json:"restrict"`
	Comment  string   `json:"comment"`
	Tags     []string `json:"tags"`
}

func getPixivCookie(ctx *fiber.Ctx) error {
	cookie, err := database.GetPixivCookie()
	if err != nil {
		log.Error().Msg(err.Error())
		return sendCommonResponse(ctx, 500, err.Error(), nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"cookie": cookie,
	})
}

type UpdateCookieRequest struct {
	Cookie string `json:"cookie"`
}

func updatePixivCookie(ctx *fiber.Ctx) error {
	req := new(UpdateCookieRequest)
	if err := ctx.BodyParser(req); err != nil {
		return sendCommonResponse(ctx, 500, err.Error(), nil)
	}
	err := database.UpdatePixivCookie(req.Cookie)
	if err != nil {
		return sendCommonResponse(ctx, 500, err.Error(), nil)
	}
	return sendCommonResponse(ctx, 200, "成功设置cookie", nil)

}

func setPixivHeaders(req *http.Request, pid string) error {
	pixivCookie, err := database.GetPixivCookie()
	if err != nil {
		return err
	}
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.0.0 Safari/537.36"
	pixivBaseURL := "https://www.pixiv.net"
	req.Header.Set("Cookie", pixivCookie)
	req.Header.Set("User-Agent", userAgent)

	if pid != "" {
		req.Header.Set("Referer", pixivBaseURL)
	}
	return nil
}

var ErrPixivNotFound = errors.New("pixiv resource not found")

func fetchPixivIllustDataFromPixiv(pid string, proxyStr string) (map[string]interface{}, error) {
	log.Debug().Str("pid", pid).Str("proxy", proxyStr).Msg("开始获取 Pixiv 数据")
	transport := &http.Transport{}
	if proxyStr != "" {
		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			log.Error().Err(err).Str("proxy_url", proxyStr).Msg("无法解析代理 URL")
			return nil, fmt.Errorf("代理配置错误: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		log.Debug().Str("proxy", proxyStr).Msg("使用代理")
	} else {
		log.Debug().Msg("不使用代理")
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	pixivURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", pid)
	req, err := http.NewRequest("GET", pixivURL, nil)
	if err != nil {
		log.Error().Err(err).Str("pid", pid).Str("url", pixivURL).Msg("构建 Pixiv 请求失败")
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	err = setPixivHeaders(req, pid)
	if err != nil {
		return nil, fmt.Errorf("设置请求头失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("pid", pid).Str("url", pixivURL).Msg("发送请求到 Pixiv 失败")
		return nil, fmt.Errorf("请求 Pixiv API 失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		log.Warn().
			Str("pid", pid).
			Int("status_code", resp.StatusCode).
			Str("response_body", string(bodyBytes)).
			AnErr("read_error", readErr).
			Msg("Pixiv API 返回了非 OK 状态")

		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrPixivNotFound
		}
		return nil, fmt.Errorf("Pixiv API 返回错误状态: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("pid", pid).Msg("读取 Pixiv 响应体失败")
		return nil, fmt.Errorf("读取 Pixiv 响应失败: %w", err)
	}
	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("pid", pid).Str("body", string(bodyBytes)).Msg("解析 Pixiv 返回的 JSON 数据失败")
		return nil, fmt.Errorf("解析 Pixiv 响应数据失败: %w", err)
	}
	bodyInterface, ok := pixivData["body"]
	if !ok {
		log.Error().Interface("response_data", pixivData).Msg("Key 'body' not found in successful Pixiv response")
		return nil, fmt.Errorf("字段 'body' 在 Pixiv 响应中不存在")
	}
	if bodyInterface == nil {
		log.Warn().Msg("Value for key 'body' is nil")
		return nil, fmt.Errorf("字段 'body' 的值为 nil")
	}
	pixivIllustData, ok := bodyInterface.(map[string]interface{})
	if !ok {
		log.Error().Interface("value", bodyInterface).Msg("Value for key 'body' is not a map[string]interface{}")
		return nil, fmt.Errorf("字段 'body' 的值不是预期的 map 结构 (实际类型: %T)", bodyInterface)
	}
	log.Debug().Msg("Successfully extracted illust data from 'body' field.")
	pidstr, err := strconv.Atoi(pid)
	exists, err := database.CheckPidExists(pidstr)
	print(exists)
	if err != nil {
		return nil, fmt.Errorf("error checking pid %d existence: %w", pid, err)
	}
	name := getIllustInformationFromPixivIllust(pixivIllustData)
	urls := getUrlsFromPixivIllust(pixivIllustData)
	bookmarkCount := getBookmarkCountFromPixivIllust(pixivIllustData)
	isBookmarked := getBookmarkFromPixivIllust(pixivIllustData)
	authorInfo := structs.Author{
		Name: getUserNameFromPixivIllust(pixivIllustData),
		UID:  getUserIdFromPixivIllust(pixivIllustData),
	}

	author, err := database.GetOrCreateAuthor(authorInfo)

	if err != nil {
		return nil, fmt.Errorf("error getting or creating author for pid %d: %w", pid, err)
	}

	if exists {
		err = database.UpdateImage(pidstr, name, author.ID, bookmarkCount, isBookmarked, urls)
		if err != nil {
			return nil, fmt.Errorf("error updating image record for pid %d: %w", pid, err)
		}
		err = database.DeleteImageTags(pidstr)
		if err != nil {
			return nil, fmt.Errorf("error clearing old tags for pid %d: %w", pid, err)
		}
	} else {
		_, err = database.CreateImage(pidstr, name, author.ID, bookmarkCount, isBookmarked, urls)
		if err != nil {
			return nil, fmt.Errorf("error creating image record for pid %d: %w", pid, err)
		}
	}

	tags := getTagsFromPixivIllust(pixivIllustData)
	for _, tagName := range tags {
		tid, err := database.GetOrCreateTagIdByName(tagName)
		if err != nil {
			return nil, fmt.Errorf("error getting or creating tag id for tag '%s' (pid %d): %w", tagName, pid, err)
		}
		err = database.InsertImageTag(pidstr, tid)
		if err != nil {
			return nil, fmt.Errorf("error inserting image-tag link for pid %d, tag id %d: %w", pid, tid, err)
		}
	}
	return pixivIllustData, nil
}

func getTagsFromPixivIllust(result map[string]interface{}) []string {
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

func getUrlsFromPixivIllust(result map[string]interface{}) structs.ImageURLs {
	var extractedUrls structs.ImageURLs
	urlsInterface, ok := result["urls"]
	if !ok {
		return extractedUrls
	}

	urlsMap, ok := urlsInterface.(map[string]interface{})
	if !ok {
		return extractedUrls
	}
	if urlValue, ok := urlsMap["original"].(string); ok {
		extractedUrls.Original = urlValue
	}
	if urlValue, ok := urlsMap["mini"].(string); ok {
		extractedUrls.Mini = urlValue
	}
	if urlValue, ok := urlsMap["thumb"].(string); ok {
		extractedUrls.Thumb = urlValue
	}
	if urlValue, ok := urlsMap["small"].(string); ok {
		extractedUrls.Small = urlValue
	}
	if urlValue, ok := urlsMap["regular"].(string); ok {
		extractedUrls.Regular = urlValue
	}

	return extractedUrls
}

func getIllustInformationFromPixivIllust(result map[string]interface{}) string {
	var illustTitle string
	illustTitle = result["illustTitle"].(string)
	return illustTitle
}
func getUserIdFromPixivIllust(result map[string]interface{}) string {
	var userId string
	userId = result["userId"].(string)
	return userId
}
func getUserNameFromPixivIllust(result map[string]interface{}) string {
	var userName string
	userName = result["userName"].(string)
	return userName
}
func getBookmarkCountFromPixivIllust(result map[string]interface{}) int {
	var bookmarkCount int
	//log.Debug().Interface("test", result).Msg("test")
	bookmarkCount = int(result["bookmarkCount"].(float64))
	return bookmarkCount
}
func getBookmarkFromPixivIllust(result map[string]interface{}) bool {
	var isBookmarked bool
	bookmarkDataValue, ok := result["bookmarkData"]
	isBookmarked = ok && bookmarkDataValue != nil
	return isBookmarked
}
func getImageByPid(ctx *fiber.Ctx) error {
	pidStr := ctx.Params("pid")
	log.Info().Str("pid", pidStr).Msg("收到 getImageByPid 请求")
	proxyStr := "http://localhost:7890"
	pixivData, err := fetchPixivIllustDataFromPixiv(pidStr, proxyStr)
	if err != nil {
		log.Warn().Err(err).Str("pid", pidStr).Msg("获取 Pixiv 数据失败")
		if errors.Is(err, ErrPixivNotFound) {
			return sendCommonResponse(ctx, fiber.StatusNotFound, "Pixiv 资源未找到", nil)
		}
		var statusCode int
		if _, scanErr := fmt.Sscanf(err.Error(), "Pixiv API 返回错误状态: %d", &statusCode); scanErr == nil {
			return sendCommonResponse(ctx, fiber.StatusBadGateway, fmt.Sprintf("Pixiv API 返回错误: %d", statusCode), nil)
		}
		if err.Error() == "设置请求头失败: 内部错误：无法获取认证信息" || err.Error() == "设置请求头失败: 内部错误：认证信息未设置" {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "服务器内部认证错误", nil)
		}
		return sendCommonResponse(ctx, fiber.StatusInternalServerError, fmt.Sprintf("处理请求时发生错误: %s", err.Error()), nil)
	}

	log.Info().Str("pid", pidStr).Msg("成功获取 Pixiv 数据，准备发送响应")
	return sendCommonResponse(ctx, http.StatusOK, "成功", pixivData)
}

func fetchPixivFollowingFromPixiv(userID string, offset int, limit int) (map[string]interface{}, error) {
	baseURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/following", userID)
	params := url.Values{}
	params.Add("offset", strconv.Itoa(offset))
	params.Add("limit", strconv.Itoa(limit))
	params.Add("rest", "show")
	params.Add("tag", "")
	params.Add("acceptingRequests", "0")
	params.Add("lang", "zh")
	fullURL := baseURL + "?" + params.Encode()

	log.Debug().Str("url", fullURL).Str("userID", userID).Msg("fetchPixivFollowingFromPixiv: Preparing request")

	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("fetchPixivFollowingFromPixiv: Failed to create request object")
		return nil, fmt.Errorf("%w: creating request: %w", ErrInternalSetupFailed, err)
	}

	err = setPixivHeaders(req, "")
	if err != nil {
		log.Error().Err(err).Msg("fetchPixivFollowingFromPixiv: Failed to set common headers")
		return nil, fmt.Errorf("%w: setting common headers: %w", ErrInternalSetupFailed, err)
	}
	req.Header.Set("Referer", fmt.Sprintf("https://www.pixiv.net/users/%s/following", userID))
	req.Header.Set("x-user-id", userID)

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("fetchPixivFollowingFromPixiv: Failed to execute request to Pixiv")
		return nil, fmt.Errorf("%w: %w", ErrPixivRequestFailed, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		log.Error().Int("status", res.StatusCode).Str("body", string(bodyBytes)).Str("userID", userID).Msg("fetchPixivFollowingFromPixiv: Pixiv returned non-OK status")
		return nil, fmt.Errorf("%w: status code %d", ErrPixivBadStatus, res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("fetchPixivFollowingFromPixiv: Failed to read response body")
		return nil, fmt.Errorf("%w: %w", ErrPixivReadBodyFailed, err)
	}

	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("body", string(bodyBytes)).Msg("fetchPixivFollowingFromPixiv: Failed to parse JSON")
		return nil, fmt.Errorf("%w: %w", ErrPixivParseFailed, err)
	}

	log.Debug().Str("userID", userID).Msg("fetchPixivFollowingFromPixiv: Successfully fetched and parsed data")
	return pixivData, nil
}

type FollowingRequestPayload struct {
	UserID string `json:"userID"`
	Offset *int   `json:"offset"`
	Limit  *int   `json:"limit"`
}

func postFollowingUsersHandler(ctx *fiber.Ctx) error {
	var payload FollowingRequestPayload
	if err := ctx.BodyParser(&payload); err != nil {
		log.Error().Err(err).Str("body", string(ctx.Body())).Msg("Handler: Cannot parse request body JSON")
		return sendCommonResponse(ctx, fiber.StatusBadRequest, "无效的请求体 JSON 格式 (Invalid request body JSON format)", nil)
	}

	if payload.UserID == "" {
		log.Error().Msg("Handler: Missing or empty userID in request body")
		return sendCommonResponse(ctx, fiber.StatusBadRequest, "请求体中必须包含有效的 userID (Request body must contain a valid userID)", nil)
	}
	userID := payload.UserID

	defaultOffset := 0
	defaultLimit := 24

	offset := defaultOffset
	if payload.Offset != nil {
		if *payload.Offset >= 0 {
			offset = *payload.Offset
		} else {
			log.Warn().Int("providedOffset", *payload.Offset).Msg("Handler: Provided offset is negative, using default")
		}
	}

	limit := defaultLimit
	if payload.Limit != nil {
		if *payload.Limit > 0 {
			limit = *payload.Limit
		} else {
			log.Warn().Int("providedLimit", *payload.Limit).Msg("Handler: Provided limit is invalid, using default")
		}
	}

	log.Info().Str("userID", userID).Int("offset", offset).Int("limit", limit).Msg("Handler: Processing request for user following list")

	pixivData, err := fetchPixivFollowingFromPixiv(userID, offset, limit)

	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Handler: Error received from fetchPixivFollowingFromPixiv")
		if errors.Is(err, ErrInternalSetupFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "内部服务器设置错误 (Internal server setup error)", nil)
		} else if errors.Is(err, ErrPixivRequestFailed) {
			return sendCommonResponse(ctx, fiber.StatusServiceUnavailable, "无法连接到 Pixiv API (Could not connect to Pixiv API)", nil)
		} else if errors.Is(err, ErrPixivBadStatus) {
			errMsg := fmt.Sprintf("Pixiv API 请求失败 (Pixiv API request failed): %v", err)
			return sendCommonResponse(ctx, fiber.StatusBadGateway, errMsg, nil)
		} else if errors.Is(err, ErrPixivReadBodyFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "读取 Pixiv 响应失败 (Failed to read Pixiv response)", nil)
		} else if errors.Is(err, ErrPixivParseFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "解析 Pixiv 响应失败 (Failed to parse Pixiv response)", nil)
		} else {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "处理请求时发生未知内部错误 (Unknown internal error processing request)", nil)
		}
	}
	log.Info().Str("userID", userID).Msg("Handler: Successfully processed request, sending response")
	return sendCommonResponse(ctx, fiber.StatusOK, "成功获取关注列表 (Successfully retrieved following list)", pixivData)
}

func fetchFollowLatestIllustsFromPixiv(page int, mode, lang, userID string) (map[string]interface{}, error) {
	//userID仅用于设置请求头
	baseURL := "https://www.pixiv.net/ajax/follow_latest/illust"
	params := url.Values{}
	params.Add("p", strconv.Itoa(page))
	params.Add("mode", mode)
	params.Add("lang", lang)
	fullURL := baseURL + "?" + params.Encode()
	log.Debug().Str("url", fullURL).Int("page", page).Str("mode", mode).Str("userID", userID).Msg("fetchFollowLatestIllusts: Preparing request")
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Msg("fetchFollowLatestIllusts: Failed to create request object")
		return nil, fmt.Errorf("%w: creating request: %w", ErrInternalSetupFailed, err)
	}

	err = setPixivHeaders(req, userID)
	if err != nil {
		log.Error().Err(err).Msg("fetchFollowLatestIllusts: Failed to set common headers")
		return nil, fmt.Errorf("%w: setting common headers: %w", ErrInternalSetupFailed, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.pixiv.net/bookmark_new_illust.php")
	req.Header.Set("x-user-id", userID)

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Str("userID", userID).Msg("fetchFollowLatestIllusts: Failed to execute request to Pixiv")
		return nil, fmt.Errorf("%w: executing request: %w", ErrPixivRequestFailed, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		log.Error().Int("status", res.StatusCode).Str("body", string(bodyBytes)).Str("url", fullURL).Str("userID", userID).Msg("fetchFollowLatestIllusts: Pixiv returned non-OK status")
		return nil, fmt.Errorf("%w: status code %d", ErrPixivBadStatus, res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Str("userID", userID).Msg("fetchFollowLatestIllusts: Failed to read response body")
		return nil, fmt.Errorf("%w: reading body: %w", ErrPixivReadBodyFailed, err)
	}

	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("body", string(bodyBytes)).Str("url", fullURL).Msg("fetchFollowLatestIllusts: Failed to parse JSON")
		return nil, fmt.Errorf("%w: unmarshaling json: %w", ErrPixivParseFailed, err)
	}

	log.Debug().Int("page", page).Str("mode", mode).Str("userID", userID).Msg("fetchFollowLatestIllusts: Successfully fetched and parsed data")
	return pixivData, nil
}

type FollowLatestRequestPayload struct {
	UserID string  `json:"userID"`
	Page   *int    `json:"page"`
	Mode   *string `json:"mode"`
	Lang   *string `json:"lang"`
}

func postFollowLatestIllustsHandler(ctx *fiber.Ctx) error {
	var payload FollowLatestRequestPayload
	if err := ctx.BodyParser(&payload); err != nil {
		log.Error().Err(err).Str("body", string(ctx.Body())).Msg("Handler: Cannot parse request body JSON for follow_latest")
		return sendCommonResponse(ctx, fiber.StatusBadRequest, "无效的请求体 JSON 格式 (Invalid request body JSON format)", nil)
	}
	if payload.UserID == "" {
		log.Error().Msg("Handler: Missing or empty userID in request body for follow_latest")
		return sendCommonResponse(ctx, fiber.StatusBadRequest, "请求体中必须包含有效的 userID (Request body must contain a valid userID)", nil)
	}
	userID := payload.UserID
	defaultPage := 1
	defaultMode := "all"
	defaultLang := "zh"

	page := defaultPage
	if payload.Page != nil {
		if *payload.Page >= 1 {
			page = *payload.Page
		} else {
			log.Warn().Int("providedPage", *payload.Page).Msg("Handler: Provided page is invalid (< 1), using default")
		}
	}
	mode := defaultMode
	if payload.Mode != nil && *payload.Mode != "" {
		mode = *payload.Mode
	} else if payload.Mode != nil {
		log.Warn().Str("providedMode", *payload.Mode).Msg("Handler: Provided mode is empty, using default")
	}
	lang := defaultLang
	if payload.Lang != nil && *payload.Lang != "" {
		lang = *payload.Lang
	} else if payload.Lang != nil {
		log.Warn().Str("providedLang", *payload.Lang).Msg("Handler: Provided lang is empty, using default")
	}
	log.Info().Str("userID", userID).Int("page", page).Str("mode", mode).Str("lang", lang).Msg("Handler: Processing request for follow latest illusts")
	pixivData, err := fetchFollowLatestIllustsFromPixiv(page, mode, lang, userID)

	if err != nil {
		log.Error().Err(err).Str("userID", userID).Int("page", page).Str("mode", mode).Str("lang", lang).Msg("Handler: Error received from fetchFollowLatestIllusts")
		if errors.Is(err, ErrInternalSetupFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "内部服务器设置错误 (Internal server setup error)", nil)
		} else if errors.Is(err, ErrPixivRequestFailed) {
			return sendCommonResponse(ctx, fiber.StatusServiceUnavailable, "无法连接到 Pixiv API (Could not connect to Pixiv API)", nil)
		} else if errors.Is(err, ErrPixivBadStatus) {
			errMsg := fmt.Sprintf("Pixiv API 请求失败 (Pixiv API request failed): %v", err)
			var statusCode int
			if _, scanErr := fmt.Sscanf(err.Error(), "Pixiv API (follow_latest) 返回错误状态: %d", &statusCode); scanErr == nil {
				if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
					return sendCommonResponse(ctx, fiber.StatusUnauthorized, "Pixiv认证失败或无权限访问关注动态", nil)
				}
			}
			return sendCommonResponse(ctx, fiber.StatusBadGateway, errMsg, nil)
		} else if errors.Is(err, ErrPixivReadBodyFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "读取 Pixiv 响应失败 (Failed to read Pixiv response)", nil)
		} else if errors.Is(err, ErrPixivParseFailed) {
			return sendCommonResponse(ctx, fiber.StatusInternalServerError, "解析 Pixiv 响应失败 (Failed to parse Pixiv response)", nil)
		}

	}

	log.Info().Str("userID", userID).Int("page", page).Str("mode", mode).Msg("Handler: Successfully processed follow_latest request, sending response")
	return sendCommonResponse(ctx, fiber.StatusOK, "成功获取关注用户的最新插画 (Successfully retrieved latest illustrations from followed users)", pixivData)
}

func fetchIllustRecommendInit(illustID string, limit int, lang string, userID string) (map[string]interface{}, error) {
	baseURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/recommend/init", illustID)
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("lang", lang)
	fullURL := baseURL + "?" + params.Encode()

	log.Debug().Str("url", fullURL).Str("illustID", illustID).Int("limit", limit).Str("lang", lang).Str("userID", userID).Msg("fetchIllustRecommendInit: Preparing request")
	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Msg("fetchIllustRecommendInit: Failed to create request object")
		return nil, fmt.Errorf("%w: creating request: %w", ErrInternalSetupFailed, err)
	}
	err = setPixivHeaders(req, userID)
	if err != nil {
		log.Error().Err(err).Msg("fetchIllustRecommendInit: Failed to set common headers")
		return nil, fmt.Errorf("%w: setting common headers: %w", ErrInternalSetupFailed, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", fmt.Sprintf("https://www.pixiv.net/artworks/%s", illustID))
	req.Header.Set("x-user-id", userID)

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Str("illustID", illustID).Str("userID", userID).Msg("fetchIllustRecommendInit: Failed to execute request to Pixiv")
		return nil, fmt.Errorf("%w: executing request: %w", ErrPixivRequestFailed, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		log.Error().Int("status", res.StatusCode).Str("body", string(bodyBytes)).Str("url", fullURL).Str("illustID", illustID).Msg("fetchIllustRecommendInit: Pixiv returned non-OK status")
		return nil, fmt.Errorf("%w: status code %d", ErrPixivBadStatus, res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("url", fullURL).Str("illustID", illustID).Msg("fetchIllustRecommendInit: Failed to read response body")
		return nil, fmt.Errorf("%w: reading body: %w", ErrPixivReadBodyFailed, err)
	}

	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("body", string(bodyBytes)).Str("url", fullURL).Msg("fetchIllustRecommendInit: Failed to parse JSON")
		return nil, fmt.Errorf("%w: unmarshaling json: %w", ErrPixivParseFailed, err)
	}

	log.Debug().Str("illustID", illustID).Int("limit", limit).Str("userID", userID).Msg("fetchIllustRecommendInit: Successfully fetched and parsed data")
	return pixivData, nil
}

func updateAllPixivImages(concurrencyLimit int) error {
	log.Info().Msg("Starting update process for all Pixiv images")

	pids, err := database.GetAllPids()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get all PIDs from database") // Structured error logging
		return fmt.Errorf("failed to get pids to update: %w", err)
	}

	if len(pids) == 0 {
		log.Info().Msg("No PIDs found in the database. Nothing to update.")
		return nil
	}

	log.Info().
		Int("pid_count", len(pids)).
		Int("concurrency_limit", concurrencyLimit).
		Msg("Found PIDs to process. Starting concurrent updates")

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrencyLimit)
	errorChan := make(chan error, len(pids))
	var errorCount int32 = 0
	var processedCount int32 = 0

	startTime := time.Now()
	for _, pid := range pids {
		wg.Add(1)
		sem <- struct{}{}

		go func(currentPid int) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Debug().Int("pid", currentPid).Msg("Processing PID")
			_, err := fetchPixivIllustDataFromPixiv(strconv.Itoa(currentPid), "http://localhost:7890")

			atomic.AddInt32(&processedCount, 1)
			currentProcessed := atomic.LoadInt32(&processedCount)

			if err != nil {
				log.Error().Err(err).Int("pid", currentPid).Msg("Error processing PID")
				errorChan <- fmt.Errorf("PID %d: %w", currentPid, err)
				atomic.AddInt32(&errorCount, 1)
			} else {
				log.Info().
					Int("pid", currentPid).
					Int32("processed_count", currentProcessed).
					Int("total_pids", len(pids)).
					Msg("Successfully processed PID")
			}
		}(pid)
	}

	log.Info().Msg("All processing jobs launched. Waiting for completion")
	wg.Wait()
	close(errorChan)
	log.Info().Msg("All processing jobs finished")

	var errorsCollected []error
	for err := range errorChan {
		errorsCollected = append(errorsCollected, err)
	}

	duration := time.Since(startTime)
	finalProcessedCount := atomic.LoadInt32(&processedCount)
	finalErrorCount := atomic.LoadInt32(&errorCount)

	log.Info().
		Dur("duration", duration).
		Int32("processed_count", finalProcessedCount).
		Int32("error_count", finalErrorCount).
		Msg("Update process finished")

	if len(errorsCollected) > 0 {
		log.Warn().Msg("Update process completed with errors")
		errorSummary := fmt.Sprintf("encountered %d errors during update:", len(errorsCollected))
		for i, e := range errorsCollected {
			if i < 10 {
				errorSummary += fmt.Sprintf("\n - %v", e)
			} else if i == 10 {
				errorSummary += "\n - ... (more errors logged above)"
				break
			}
		}
		return errors.New(errorSummary)
	}

	log.Info().Msg("Successfully processed all PIDs without errors.")
	return nil
}

func triggerUpdateAllHandler(c *fiber.Ctx) error {
	limit := 1

	log.Info().Int("concurrency_limit", limit).Msg("Received request to trigger Pixiv update process")
	go func(requestedLimit int) {
		log.Info().Int("concurrency_limit", requestedLimit).Msg("Starting background Pixiv update task...")
		err := updateAllPixivImages(requestedLimit)
		if err != nil {
			log.Error().Err(err).Msg("Background Pixiv update task finished with error")
		} else {
			log.Info().Msg("Background Pixiv update task finished successfully")
		}
	}(limit)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":           "Pixiv image update process initiated.",
		"concurrency_limit": limit,
		"status_check_info": "Check server logs for progress and completion status.",
		"timestamp":         time.Now(),
	})
}

func updateAllPixivImagesChecker(concurrencyLimit int) error {
	// 该函数仅用于再次查询bookmark为0的画作重新获取，避免因网络问题没获取到图片

	log.Info().Msg("Starting update process for all Pixiv images")

	pids, err := database.GetPidsByBookmarkRange(0, 0)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get all PIDs from database") // Structured error logging
		return fmt.Errorf("failed to get pids to update: %w", err)
	}

	if len(pids) == 0 {
		log.Info().Msg("No PIDs found in the database. Nothing to update.")
		return nil
	}

	log.Info().
		Int("pid_count", len(pids)).
		Int("concurrency_limit", concurrencyLimit).
		Msg("Found PIDs to process. Starting concurrent updates")

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrencyLimit)
	errorChan := make(chan error, len(pids))
	var errorCount int32 = 0
	var processedCount int32 = 0

	startTime := time.Now()
	for _, pid := range pids {
		wg.Add(1)
		sem <- struct{}{}

		go func(currentPid int) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Debug().Int("pid", currentPid).Msg("Processing PID")
			_, err := fetchPixivIllustDataFromPixiv(strconv.Itoa(currentPid), "http://localhost:7890")

			atomic.AddInt32(&processedCount, 1)
			currentProcessed := atomic.LoadInt32(&processedCount)

			if err != nil {
				log.Error().Err(err).Int("pid", currentPid).Msg("Error processing PID")
				errorChan <- fmt.Errorf("PID %d: %w", currentPid, err)
				atomic.AddInt32(&errorCount, 1)
			} else {
				log.Info().
					Int("pid", currentPid).
					Int32("processed_count", currentProcessed).
					Int("total_pids", len(pids)).
					Msg("Successfully processed PID")
			}
		}(pid)
	}

	log.Info().Msg("All processing jobs launched. Waiting for completion")
	wg.Wait()
	close(errorChan)
	log.Info().Msg("All processing jobs finished")

	var errorsCollected []error
	for err := range errorChan {
		errorsCollected = append(errorsCollected, err)
	}

	duration := time.Since(startTime)
	finalProcessedCount := atomic.LoadInt32(&processedCount)
	finalErrorCount := atomic.LoadInt32(&errorCount)

	log.Info().
		Dur("duration", duration).
		Int32("processed_count", finalProcessedCount).
		Int32("error_count", finalErrorCount).
		Msg("Update process finished")

	if len(errorsCollected) > 0 {
		log.Warn().Msg("Update process completed with errors")
		errorSummary := fmt.Sprintf("encountered %d errors during update:", len(errorsCollected))
		for i, e := range errorsCollected {
			if i < 10 {
				errorSummary += fmt.Sprintf("\n - %v", e)
			} else if i == 10 {
				errorSummary += "\n - ... (more errors logged above)"
				break
			}
		}
		return errors.New(errorSummary)
	}

	log.Info().Msg("Successfully processed all PIDs without errors.")
	return nil
}

func triggerUpdateAllHandlerChecker(c *fiber.Ctx) error {
	limit := 1

	log.Info().Int("concurrency_limit", limit).Msg("Received request to trigger Pixiv update process")
	go func(requestedLimit int) {
		log.Info().Int("concurrency_limit", requestedLimit).Msg("Starting background Pixiv update task...")
		err := updateAllPixivImagesChecker(requestedLimit)
		if err != nil {
			log.Error().Err(err).Msg("Background Pixiv update task finished with error")
		} else {
			log.Info().Msg("Background Pixiv update task finished successfully")
		}
	}(limit)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":           "Pixiv image update process initiated.",
		"concurrency_limit": limit,
		"status_check_info": "Check server logs for progress and completion status.",
		"timestamp":         time.Now(),
	})
}
