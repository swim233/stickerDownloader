package handler

import (
    "bytes"
    "image"
    "image/png"

    _ "golang.org/x/image/webp" // 导入 WebP 解码器
)

type formatConverter struct {
}

func (f formatConverter) convertWebPToPNG(webp []byte) ([]byte, error) {
    // 将 WebP 数据加载到内存中
    reader := bytes.NewReader(webp)

    // 解码 WebP 图像
    img, _, err := image.Decode(reader)
    if err != nil {
        return nil, err
    }

    // 创建一个缓冲区用于存储 PNG 数据
    var buffer bytes.Buffer

    // 将图像编码为 PNG 格式
    err = png.Encode(&buffer, img)
    if err != nil {
        return nil, err
    }

    // 返回 PNG 数据
    return buffer.Bytes(), nil
}
