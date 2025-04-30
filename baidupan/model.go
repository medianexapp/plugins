package main

const (
	BaiduPanURL = "https://pan.baidu.com"
)

type QrcodeData struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUrl string `json:"verification_url"`
	QrcodeURL       string `json:"qrcode_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type Response struct {
	Errno     int    `json:"errno"`
	ErrMsg    string `json:"errmsg"`
	RequestId string `json:"request_id"`
}

// /
type UserInfo struct {
	BaiduName   string `json:"baidu_name"`
	NetdiskName string `json:"netdisk_name"`
	Uk          int    `json:"uk"`
	VipType     int    `json:"vip_type"`
}

type FileListItem struct {
	FsId           uint64 `json:"fs_id"`
	Path           string `json:"path"`
	ServerFilename string `json:"server_filename"`
	Size           uint64 `json:"size"`
	ServerMtime    uint64 `json:"server_mtime"`
	ServerAtime    uint64 `json:"server_atime"`
	ServerCtime    uint64 `json:"server_ctime"`
	IsDir          uint64 `json:"isdir"`
	Category       uint64 `json:"category"`
}

type FileListResponse struct {
	List []*FileListItem `json:"list"`
}

type FileMetasRequest struct {
	FsIds     []uint64 `query:"fsids"`
	Dlink     int      `query:"dlink"`     // 1
	Thumb     int      `query:"thumb"`     // 1
	NeedMedia int      `query:"needmedia"` // 1
	Detail    int      `query:"detail"`    // 1
}

type FileMetaItem struct {
	Category    int    `json:"category"`
	DateTaken   int    `json:"date_taken"`
	Dlink       string `json:"dlink"` // expire 8H
	Filename    string `json:"filename"`
	FsID        int64  `json:"fs_id"`
	Height      int    `json:"height"`
	Isdir       int    `json:"isdir"`
	Md5         string `json:"md5"`
	OperID      int    `json:"oper_id"`
	Path        string `json:"path"`
	ServerCtime int    `json:"server_ctime"`
	ServerMtime int    `json:"server_mtime"`
	Size        int    `json:"size"`
	Thumbs      struct {
		Icon string `json:"icon"`
		URL1 string `json:"url1"`
		URL2 string `json:"url2"`
		URL3 string `json:"url3"`
	} `json:"thumbs"`
	Width int `json:"width"`
}

type FileMetasResponse struct {
	List []*FileMetaItem `json:"list"`
}
