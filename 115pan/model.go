package main

// https://www.yuque.com/115yun/open/um8whr91bxb5997o

const (
	Api115PanAddr = "https://proapi.115.com/"
)

type QrResponse struct {
	State   int    `json:"state"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Error   string `json:"error"`
	Errno   int    `json:"errno"`
}

type Response struct {
	State   any         `json:"state"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
	Errno   int         `json:"errno"`
}

type AuthDeviceCodeData struct {
	Uid    string `json:"uid"`
	Time   int64  `json:"time"`
	Qrcode string `json:"qrcode"`
	Sign   string `json:"sign"`
}
type UserInfo struct {
	UserId   int    `json:"user_id"`
	UserName string `json:"user_name"`
}

type FileEntry struct {
	Fid string `json:"fid"` // if fid is empty,show /
	Aid string `json:"aid"` // 文件的状态，aid 的别名。1 正常，7 删除(回收站)，120 彻底删除
	Pid string `json:"pid"` // 父目录ID
	Fc  string `json:"fc"`  // 0 folder 1 file
	Fn  string `json:"fn"`  // file name
	Pc  string `json:"pc"`  // file pick code
	// Isp string `json:"isp"` // passwd

	Upt  int    `json:"upt"`  // 修改时间
	Uppt int    `json:"uppt"` // 上传时间
	Fs   uint64 `json:"fs"`   // file size

	Fta string `json:"fta"` // 文件状态 0/2 未上传完成，1 已上传完成
}

type FileURL struct {
	FileName string `json:"file_name"`
	FileSize uint64 `json:"file_size"`
	PickCode string `json:"pick_code"`
	Sha1     string `json:"sha1"`
	Url      struct {
		Url string `json:"url"`
	} `json:"url"`
}

type SubtitleData struct {
	List []Subtitle `json:"list"`
}

type Subtitle struct {
	Sid      string `json:"sid"`
	Language string `json:"language"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Type     string `json:"type"`
}

type PlayVideoInfo struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
	VideoURL []struct {
		URL         string `json:"url"`
		Definition  int    `json:"definition"`
		DefinitionN int    `json:"definition_n"`
		Title       string `json:"title"`
	} `json:"video_url"`

	// TODO add audio
	// MultitrackList []struct {
	// 	Title string `json:"title"`
	// }
}
