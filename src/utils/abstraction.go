package utils

type Format string

func (f Format) String() string {
	switch f {
	case JpegFormat:
		return "jpeg"
	case PngFormat:
		return "png"
	case WebpFormat:
		return "webp"
	default:
		return string(f)
	}
}

const (
	JpegFormat Format = "jpeg"
	PngFormat  Format = "png"
	WebpFormat Format = "webp"
)
