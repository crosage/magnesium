package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
)

func getTagsFromPid(pid string) (map[string]interface{}, error) {
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	url := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", pid)
	fmt.Println(url)
	method := "GET"
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
	}
	req, err := http.NewRequest(method, url, nil)
	fmt.Println("*************")
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", "PHPSESSID=42279487_ixsJf0c7B8Uko3o71Nx8HG0mxg0lrhZo; __cf_bm=GwbgPAFqBwYBMw.02LRDMBquapFwJBKkS232J_w34Hc-1716448255-1.0.1.1-Blm.czHdnfeW0in2mV0UnkFmsKcNO5iUURs8YrHjDXAvGlHW1B5HIoKVH2zQBoGCW6E0frILPI0GWKer8SaxNdsmzN7m4Xd2CijF5hmlhMY; a_type=0; b_type=0; c_type=31; privacy_policy_notification=0; yuid_b=OQQ0UgA")
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
	return response, nil
}
func pixivHandler(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	result, err := getTagsFromPid(id)
	body, ok := result["body"].(map[string]interface{})
	if !ok {
		return errors.New("未找到body字段或者类型不正确")
	}

	tags, ok := body["tags"].(map[string]interface{})
	if !ok {
		return errors.New("未找到tags字段或者类型不正确")
	}

	tagList, ok := tags["tags"].([]interface{})
	if !ok {
		return errors.New("未找到tags字段或者类型不正确")
	}
	fmt.Println(tagList)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	return nil
}
