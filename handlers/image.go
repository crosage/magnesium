package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"go_/database"
	"go_/structs"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

func getInformationFromPid(pid int) (map[string]interface{}, error) {
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	url := "https://www.pixiv.net/ajax/illust/" + strconv.Itoa(pid)
	fmt.Println(url)
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
	return response["body"].(map[string]interface{}), nil
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
func pixivHandler(ctx *fiber.Ctx) error {
	pidStr := ctx.Params("id")
	pid, err := strconv.Atoi(pidStr)
	result, err := getInformationFromPid(pid)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	name := getIllustInformationFromResult(result)
	author := structs.Author{
		Name: getUserNameFromResult(result),
		UID:  getUserIdFromResult(result),
	}
	author, err = database.GetOrCreateAuthor(author)
	_, err = database.CreateImage(pid, name, "", author.ID)
	tags := getTagsFromResult(result)
	for _, tag := range tags {
		tid, err := database.GetorCreateTagIdByName(tag)
		if err != nil {
			return sendCommonResponse(ctx, 500, "", nil)
		}
		err = database.InsertImageTag(pid, tid)
	}
	fmt.Println(tags)
	return sendCommonResponse(ctx, 200, "", nil)
}
