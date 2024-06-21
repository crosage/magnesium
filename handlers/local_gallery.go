package handlers

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	jsoniter "github.com/json-iterator/go"
	"go_/database"
	"go_/structs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

func getAllGalleries(ctx *fiber.Ctx) error {
	var galleries []structs.LocalGallery
	galleries, err := database.GetAllGalleries()
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", map[string]interface{}{
		"total":     len(galleries),
		"galleries": galleries,
	})
}

func createGallery(ctx *fiber.Ctx) error {
	var gallery structs.LocalGallery
	err := jsoniter.Unmarshal(ctx.Body(), &gallery)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	err = database.CreateLocalGalleryPath(gallery.Path)
	if err != nil {
		return sendCommonResponse(ctx, 500, "", nil)
	}
	return sendCommonResponse(ctx, 200, "成功", nil)
}

func deleteGallery(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := strconv.Atoi(idParam)
	err := database.DeleteLocalGalleryByID(id)
	if err != nil {
		return sendCommonResponse(ctx, 500, "删除失败", nil)
	}
	return sendCommonResponse(ctx, 200, "删除成功", nil)
}

func initGallery(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := strconv.Atoi(idParam)
	folderPath, err := database.GetGalleryById(id)
	err = filepath.Walk(folderPath.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			re := regexp.MustCompile(`(\d+)_p(\d+)\.(\w+)`)
			match := re.FindStringSubmatch(info.Name())
			if match != nil {
				pid, _ := strconv.Atoi(match[1])
				pageId, _ := strconv.Atoi(match[2])
				fileType := match[3]
				err := pixivHandler(pid, folderPath.Path, fileType)
				if err != nil {
					return sendCommonResponse(ctx, 500, "爬虫过程出现错误", nil)
				}
				_, err = database.InsertPageByPid(pid, pageId)
				if err != nil {
					return sendCommonResponse(ctx, 500, "爬虫过程出现错误", nil)
				}
			}
			fmt.Println(info.Name())
		}
		return nil
	})
	if err != nil {
		return sendCommonResponse(ctx, 500, "遍历过程出现错误", nil)
	}
	return sendCommonResponse(ctx, 200, "完成初始化", nil)
}
