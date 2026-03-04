package lib

import "strconv"

type DownloadRequest struct {
	ChatID            int64
	MessageID         int
	CallbackMessageID int
	DownloadType      string
	DownloadFormat    string
}

type DownloadRequestID string

func GetRequestID(chatID int64, messageID int) DownloadRequestID {

	strChatID := strconv.Itoa(int(chatID))
	strMsgID := strconv.Itoa(messageID)

	return DownloadRequestID(strChatID + strMsgID)
}
