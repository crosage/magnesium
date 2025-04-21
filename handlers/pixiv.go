package handlers

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"go_/database"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

func getImageByPid(ctx *fiber.Ctx) error {
	var err error
	pidStr := ctx.Params("pid")
	log.Log().Msg("请求 PID (原始数据代理): " + pidStr)
	proxyStr := "http://localhost:7890"
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		log.Fatal().Err(err).Str("proxy_url", proxyStr).Msg("无法解析代理 URL")
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
	pixivURL := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", pidStr)
	req, err := http.NewRequest("GET", pixivURL, nil)

	if err != nil {
		log.Error().Err(err)
		return sendCommonResponse(ctx, 500, "构建请求失败", nil)
	}
	err = setPixivHeaders(req, pidStr)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("pid", pidStr).Msg("发送请求到 Pixiv 失败")
		return sendCommonResponse(ctx, fiber.StatusServiceUnavailable, "请求 Pixiv API 失败", nil)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Error().
			Str("pid", pidStr).
			Int("status_code", resp.StatusCode).
			Str("response_body", string(bodyBytes)).
			Msg("Pixiv 返回了非 OK 状态")
		if resp.StatusCode == http.StatusNotFound {
			return sendCommonResponse(ctx, fiber.StatusNotFound, "Pixiv 资源未找到或 API 错误", nil)
		}
		return sendCommonResponse(ctx, fiber.StatusBadGateway, fmt.Sprintf("Pixiv API 返回错误状态: %d", resp.StatusCode), nil)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Str("pid", pidStr).Msg("读取 Pixiv 响应体失败")
		return sendCommonResponse(ctx, fiber.StatusInternalServerError, "读取 Pixiv 响应失败", nil)
	}
	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("pid", pidStr).Str("body", string(bodyBytes)).Msg("解析 Pixiv 返回的 JSON 数据失败")
		return sendCommonResponse(ctx, fiber.StatusInternalServerError, "解析 Pixiv 响应数据失败", nil)
	}
	log.Info().Str("pid", pidStr).Msg("成功获取 Pixiv 数据，通过 sendCommonResponse 代理")
	return sendCommonResponse(ctx, http.StatusOK, "成功", pixivData)
}

func fetchPixivFollowing(userID string, offset int, limit int) (map[string]interface{}, error) {
	baseURL := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/following", userID)
	params := url.Values{}
	params.Add("offset", strconv.Itoa(offset))
	params.Add("limit", strconv.Itoa(limit))
	params.Add("rest", "show")
	params.Add("tag", "")
	params.Add("acceptingRequests", "0")
	params.Add("lang", "zh")
	fullURL := baseURL + "?" + params.Encode()

	log.Debug().Str("url", fullURL).Str("userID", userID).Msg("fetchPixivFollowing: Preparing request")

	proxyURL, _ := url.Parse("http://127.0.0.1:7890")
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("fetchPixivFollowing: Failed to create request object")
		return nil, fmt.Errorf("%w: creating request: %w", ErrInternalSetupFailed, err)
	}

	err = setPixivHeaders(req, "")
	if err != nil {
		log.Error().Err(err).Msg("fetchPixivFollowing: Failed to set common headers")
		return nil, fmt.Errorf("%w: setting common headers: %w", ErrInternalSetupFailed, err)
	}
	req.Header.Set("Referer", fmt.Sprintf("https://www.pixiv.net/users/%s/following", userID))
	req.Header.Set("x-user-id", userID)

	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("fetchPixivFollowing: Failed to execute request to Pixiv")
		return nil, fmt.Errorf("%w: %w", ErrPixivRequestFailed, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		log.Error().Int("status", res.StatusCode).Str("body", string(bodyBytes)).Str("userID", userID).Msg("fetchPixivFollowing: Pixiv returned non-OK status")
		return nil, fmt.Errorf("%w: status code %d", ErrPixivBadStatus, res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("fetchPixivFollowing: Failed to read response body")
		return nil, fmt.Errorf("%w: %w", ErrPixivReadBodyFailed, err)
	}

	var pixivData map[string]interface{}
	err = jsoniter.Unmarshal(bodyBytes, &pixivData)
	if err != nil {
		log.Error().Err(err).Str("body", string(bodyBytes)).Msg("fetchPixivFollowing: Failed to parse JSON")
		return nil, fmt.Errorf("%w: %w", ErrPixivParseFailed, err)
	}

	log.Debug().Str("userID", userID).Msg("fetchPixivFollowing: Successfully fetched and parsed data")
	return pixivData, nil
}

type FollowingRequestPayload struct {
	UserID string `json:"userID"`
	Offset *int   `json:"offset"`
	Limit  *int   `json:"limit"`
}

func PostFollowingUsersHandler(ctx *fiber.Ctx) error {
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

	pixivData, err := fetchPixivFollowing(userID, offset, limit)

	if err != nil {
		log.Error().Err(err).Str("userID", userID).Msg("Handler: Error received from fetchPixivFollowing")
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
