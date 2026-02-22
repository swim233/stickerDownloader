package lib

var TranslationsMap map[string]Translations

type Translations struct {
	CurrentStickerSet        string `json:"CurrentStickerSet"`
	PickDownloadMethod       string `json:"PickDownloadMethod"`
	DownloadSingleSticker    string `json:"DownloadSingleSticker"`
	DownloadStickerPack      string `json:"DownloadStickerPack"`
	DownloadingSingleSticker string `json:"DownloadingSingleSticker"`
	PickDownloadFormat       string `json:"PickDownloadFormat"`
	DownloadingStickerSet    string `json:"DownloadingStickerSet"`
	StickerSetIsNull         string `json:"StickerSetIsNull"`
	Help                     string `json:"Help"`
	Cancel                   string `json:"Cancel"`
	SuccessChangeLanguage    string `json:"SuccessChangeLanguage"`
}
