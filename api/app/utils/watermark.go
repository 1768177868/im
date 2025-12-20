package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"

	"github.com/goravel/framework/facades"
)

// AddWatermark 为图片添加图片水印
// imgData: 原始图片数据
// watermarkImagePath: 水印图片路径（相对于存储根目录）
// position: 水印位置 (top-left, top-right, bottom-left, bottom-right, center)
// opacity: 透明度 (0-255, 255为完全不透明)
// scale: 水印缩放比例 (0.1-1.0, 1.0为原始大小)
func AddWatermark(imgData []byte, watermarkImagePath, position string, opacity int, scale float64) ([]byte, error) {
	if watermarkImagePath == "" {
		return imgData, nil
	}

	// 解码原始图片
	img, format, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	// 只处理 PNG 和 JPEG
	if format != "png" && format != "jpeg" {
		return imgData, nil
	}

	// 加载水印图片
	storage := facades.Storage().Disk("local")
	watermarkData, err := storage.Get(watermarkImagePath)
	if err != nil {
		facades.Log().Errorf("加载水印图片失败: %v, path: %s", err, watermarkImagePath)
		return imgData, nil
	}

	watermarkImg, _, err := image.Decode(bytes.NewReader([]byte(watermarkData)))
	if err != nil {
		facades.Log().Errorf("解码水印图片失败: %v", err)
		return imgData, nil
	}

	// 创建新的图片（RGBA格式）
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, image.Point{}, draw.Src)

	// 计算水印尺寸（根据缩放比例）
	watermarkBounds := watermarkImg.Bounds()
	watermarkWidth := int(float64(watermarkBounds.Dx()) * scale)
	watermarkHeight := int(float64(watermarkBounds.Dy()) * scale)

	// 如果缩放后尺寸为0，使用原始尺寸
	if watermarkWidth <= 0 {
		watermarkWidth = watermarkBounds.Dx()
	}
	if watermarkHeight <= 0 {
		watermarkHeight = watermarkBounds.Dy()
	}

	// 缩放水印图片
	scaledWatermark := image.NewRGBA(image.Rect(0, 0, watermarkWidth, watermarkHeight))
	// 使用简单的最近邻缩放
	for y := 0; y < watermarkHeight; y++ {
		for x := 0; x < watermarkWidth; x++ {
			srcX := x * watermarkBounds.Dx() / watermarkWidth
			srcY := y * watermarkBounds.Dy() / watermarkHeight
			scaledWatermark.Set(x, y, watermarkImg.At(srcX, srcY))
		}
	}

	// 计算水印位置
	watermarkX, watermarkY := calculateWatermarkImagePosition(
		bounds,
		watermarkWidth,
		watermarkHeight,
		position,
	)

	// 创建带透明度的水印图片
	watermarkRGBA := image.NewRGBA(image.Rect(0, 0, watermarkWidth, watermarkHeight))
	draw.Draw(watermarkRGBA, watermarkRGBA.Bounds(), scaledWatermark, image.Point{}, draw.Src)

	// 应用透明度并绘制水印
	drawWatermarkWithOpacity(rgba, watermarkRGBA, watermarkX, watermarkY, opacity)

	// 编码图片
	var buf bytes.Buffer
	if format == "png" {
		err = png.Encode(&buf, rgba)
	} else {
		err = jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 90})
	}
	if err != nil {
		return nil, fmt.Errorf("编码图片失败: %w", err)
	}

	return buf.Bytes(), nil
}

// calculateWatermarkImagePosition 计算水印图片位置
func calculateWatermarkImagePosition(bounds image.Rectangle, watermarkWidth, watermarkHeight int, position string) (x, y int) {
	width := bounds.Dx()
	height := bounds.Dy()
	margin := 20

	switch position {
	case "top-left":
		x = margin
		y = margin
	case "top-right":
		x = width - watermarkWidth - margin
		y = margin
	case "bottom-left":
		x = margin
		y = height - watermarkHeight - margin
	case "bottom-right":
		x = width - watermarkWidth - margin
		y = height - watermarkHeight - margin
	case "center":
		x = (width - watermarkWidth) / 2
		y = (height - watermarkHeight) / 2
	default:
		// 默认右下角
		x = width - watermarkWidth - margin
		y = height - watermarkHeight - margin
	}

	return x, y
}

// drawWatermarkWithOpacity 绘制带透明度的水印
func drawWatermarkWithOpacity(dst *image.RGBA, watermark *image.RGBA, x, y int, opacity int) {
	opacityFloat := float64(opacity) / 255.0
	watermarkBounds := watermark.Bounds()
	dstBounds := dst.Bounds()

	for wy := 0; wy < watermarkBounds.Dy(); wy++ {
		for wx := 0; wx < watermarkBounds.Dx(); wx++ {
			dstX := x + wx
			dstY := y + wy

			// 检查是否超出目标图片范围
			if dstX < 0 || dstX >= dstBounds.Dx() || dstY < 0 || dstY >= dstBounds.Dy() {
				continue
			}

			// 获取水印像素和目标像素
			watermarkColor := watermark.RGBAAt(wx, wy)
			dstColor := dst.RGBAAt(dstX, dstY)

			// 计算混合后的颜色（使用 alpha 混合）
			alpha := float64(watermarkColor.A) * opacityFloat / 255.0
			invAlpha := 1.0 - alpha

			r := uint8(float64(watermarkColor.R)*alpha + float64(dstColor.R)*invAlpha)
			g := uint8(float64(watermarkColor.G)*alpha + float64(dstColor.G)*invAlpha)
			b := uint8(float64(watermarkColor.B)*alpha + float64(dstColor.B)*invAlpha)
			a := uint8(float64(watermarkColor.A)*opacityFloat + float64(dstColor.A)*invAlpha)

			dst.SetRGBA(dstX, dstY, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
}
