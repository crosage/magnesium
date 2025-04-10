package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"go_/database"
	"io"
	"net/http"
	"net/url"
	"time"
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
