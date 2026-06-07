package main

import (
	"fmt"
)

// SystemInfoResponse represents the Emby /System/Info endpoint response
type SystemInfoResponse struct {
	SystemUpdateLevel string `json:"SystemUpdateLevel"`
	OperatingSystem   string `json:"OperatingSystem"`
	Version           string `json:"Version"`
	Id                string `json:"Id"`
	ServerName        string `json:"ServerName"`
}

// AuthByNameResponse represents the Emby /Users/AuthenticateByName endpoint response
type AuthByNameResponse struct {
	User        *AuthByNameUser `json:"User"`
	SessionInfo interface{}     `json:"SessionInfo"`
	AccessToken string          `json:"AccessToken"`
	ServerId    string          `json:"ServerId"`
}

// AuthByNameUser represents the user info in AuthenticateByName response
type AuthByNameUser struct {
	Name                  string `json:"Name"`
	Id                    string `json:"Id"`
	ServerId              string `json:"ServerId"`
	HasPassword           bool   `json:"HasPassword"`
	HasConfiguredPassword bool   `json:"HasConfiguredPassword"`
}

// ItemsResponse represents the Emby /Items endpoint response
type ItemsResponse struct {
	Items            []*EmbyItem `json:"Items"`
	TotalRecordCount int         `json:"TotalRecordCount"`
	StartIndex       int         `json:"StartIndex"`
}

// Item represents a genre reference in an Emby response
type Item struct {
	Name            string      `json:"Name"`
	Id              interface{} `json:"Id"`
	Role            string      `json:"Role"`
	PrimaryImageTag string      `json:"PrimaryImageTag"`
}

// EmbyItem represents a single item from the Emby API
type EmbyItem struct {
	Name     string `json:"Name"`
	Id       string `json:"Id"`
	Type     string `json:"Type"`
	IsFolder bool   `json:"IsFolder"`
	Path     string `json:"Path"`
	ParentId string `json:"ParentId"`
	// CollectionType string            `json:"CollectionType"`
	Overview       string `json:"Overview"`
	RunTimeTicks   int64  `json:"RunTimeTicks"`
	ProductionYear int    `json:"ProductionYear"`
	// season index
	IndexNumber int    `json:"IndexNumber"`
	SeriesName  string `json:"SeriesName"`
	SeriesId    string `json:"SeriesId"`
	SeasonId    string `json:"SeasonId"`
	SeasonName  string `json:"SeasonName"`

	DateCreated    string  `json:"DateCreated"`
	GenreItems     []*Item `json:"GenreItems"`
	Studios        []*Item `json:"Studios"`
	OfficialRating string  `json:"OfficialRating"`
	OriginalTitle  string  `json:"OriginalTitle"`
	People         []*Item `json:"Poople"`

	ImageTags             map[string]string `json:"ImageTags"`
	BackdropImageTags     []string          `json:"BackdropImageTags"`
	SeriesPrimaryImageTag string            `json:"SeriesPrimaryImageTag"`

	CollectionType string `json:"CollectionType"`
}

// UserResponse represents the /Users endpoint response (list of users)
type UserResponse struct {
	Name       string `json:"Name"`
	Id         string `json:"Id"`
	ServerId   string `json:"ServerId"`
	ServerName string `json:"ServerName"`
}

type FilterItem struct {
	Name string `json:"Name"`
	Id   string `json:"Id"`
}

// FilterItemsResponse represents the /Genres endpoint response
type FilterItemsResponse struct {
	Items            []*FilterItem `json:"Items"`
	TotalRecordCount int           `json:"TotalRecordCount"`
}

type ErrResponse struct {
	ErrorCode string `json:"ErrorCode"`
	Message   string `json:"Message"`
}

func (e ErrResponse) Error() string {
	return fmt.Sprintf("%s(%s)", e.Message, e.ErrorCode)
}

type MediaSourcesData struct {
	MediaSources []struct {
		Protocol             string `json:"Protocol"`
		ID                   string `json:"Id"`
		Path                 string `json:"Path"`
		Type                 string `json:"Type"`
		Container            string `json:"Container"`
		Size                 int    `json:"Size"`
		Name                 string `json:"Name"`
		IsRemote             bool   `json:"IsRemote"`
		RunTimeTicks         int64  `json:"RunTimeTicks"`
		SupportsTranscoding  bool   `json:"SupportsTranscoding"`
		SupportsDirectStream bool   `json:"SupportsDirectStream"`
		SupportsDirectPlay   bool   `json:"SupportsDirectPlay"`
		IsInfiniteStream     bool   `json:"IsInfiniteStream"`
		RequiresOpening      bool   `json:"RequiresOpening"`
		RequiresClosing      bool   `json:"RequiresClosing"`
		RequiresLooping      bool   `json:"RequiresLooping"`
		SupportsProbing      bool   `json:"SupportsProbing"`
		MediaStreams         []struct {
			Codec                           string `json:"Codec"`
			CodecTag                        string `json:"CodecTag"`
			Language                        string `json:"Language"`
			TimeBase                        string `json:"TimeBase"`
			VideoRange                      string `json:"VideoRange,omitempty"`
			DisplayTitle                    string `json:"DisplayTitle"`
			IsInterlaced                    bool   `json:"IsInterlaced"`
			BitRate                         int    `json:"BitRate"`
			BitDepth                        int    `json:"BitDepth,omitempty"`
			RefFrames                       int    `json:"RefFrames,omitempty"`
			IsDefault                       bool   `json:"IsDefault"`
			IsForced                        bool   `json:"IsForced"`
			IsHearingImpaired               bool   `json:"IsHearingImpaired"`
			Height                          int    `json:"Height,omitempty"`
			Width                           int    `json:"Width,omitempty"`
			AverageFrameRate                int    `json:"AverageFrameRate,omitempty"`
			RealFrameRate                   int    `json:"RealFrameRate,omitempty"`
			Profile                         string `json:"Profile"`
			Type                            string `json:"Type"`
			AspectRatio                     string `json:"AspectRatio,omitempty"`
			Index                           int    `json:"Index"`
			IsExternal                      bool   `json:"IsExternal"`
			IsTextSubtitleStream            bool   `json:"IsTextSubtitleStream"`
			SupportsExternalStream          bool   `json:"SupportsExternalStream"`
			Protocol                        string `json:"Protocol"`
			PixelFormat                     string `json:"PixelFormat,omitempty"`
			Level                           int    `json:"Level,omitempty"`
			IsAnamorphic                    bool   `json:"IsAnamorphic,omitempty"`
			ExtendedVideoType               string `json:"ExtendedVideoType"`
			ExtendedVideoSubType            string `json:"ExtendedVideoSubType"`
			ExtendedVideoSubTypeDescription string `json:"ExtendedVideoSubTypeDescription"`
			AttachmentSize                  int    `json:"AttachmentSize"`
			ChannelLayout                   string `json:"ChannelLayout,omitempty"`
			Channels                        int    `json:"Channels,omitempty"`
			SampleRate                      int    `json:"SampleRate,omitempty"`
		} `json:"MediaStreams"`
		Formats             []any `json:"Formats"`
		Bitrate             int   `json:"Bitrate"`
		RequiredHTTPHeaders struct {
		} `json:"RequiredHttpHeaders"`
		DirectStreamURL         string `json:"DirectStreamUrl"`
		ReadAtNativeFramerate   bool   `json:"ReadAtNativeFramerate"`
		DefaultAudioStreamIndex int    `json:"DefaultAudioStreamIndex"`
	} `json:"MediaSources"`
	PlaySessionID string `json:"PlaySessionId"`
}

var defaultPlayBackProfile = `{"DeviceProfile":{"MaxStaticBitrate":140000000,"MaxStreamingBitrate":140000000,"MusicStreamingTranscodingBitrate":192000,"DirectPlayProfiles":[{"Container":"mp4,m4v","Type":"Video","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"mkv","Type":"Video","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"flv","Type":"Video","VideoCodec":"h264","AudioCodec":"aac,mp3"},{"Container":"3gp","Type":"Video","VideoCodec":"","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"mov","Type":"Video","VideoCodec":"h264","AudioCodec":"mp3,aac,opus,flac,vorbis"},{"Container":"opus","Type":"Audio"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3"},{"Container":"mp2,mp3","Type":"Audio","AudioCodec":"mp2"},{"Container":"aac","Type":"Audio","AudioCodec":"aac"},{"Container":"m4a","AudioCodec":"aac","Type":"Audio"},{"Container":"mp4","AudioCodec":"aac","Type":"Audio"},{"Container":"flac","Type":"Audio"},{"Container":"webma,webm","Type":"Audio"},{"Container":"wav","Type":"Audio","AudioCodec":"PCM_S16LE,PCM_S24LE"},{"Container":"ogg","Type":"Audio"},{"Container":"webm","Type":"Video","AudioCodec":"vorbis,opus","VideoCodec":"av1,VP8,VP9"}],"TranscodingProfiles":[{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Streaming","Protocol":"hls","MaxAudioChannels":"2","MinSegments":"1","BreakOnNonKeyFrames":false},{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"opus","Type":"Audio","AudioCodec":"opus","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"wav","Type":"Audio","AudioCodec":"wav","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"opus","Type":"Audio","AudioCodec":"opus","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp3","Type":"Audio","AudioCodec":"mp3","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"aac","Type":"Audio","AudioCodec":"aac","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"wav","Type":"Audio","AudioCodec":"wav","Context":"Static","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mkv","Type":"Video","AudioCodec":"mp3,aac,opus,flac,vorbis","VideoCodec":"h264,h265,hevc,av1,vp8,vp9","Context":"Static","MaxAudioChannels":"2","CopyTimestamps":true},{"Container":"m4s,ts","Type":"Video","AudioCodec":"mp3,aac","VideoCodec":"h264,h265,hevc,av1","Context":"Streaming","Protocol":"hls","MaxAudioChannels":"2","MinSegments":"1","BreakOnNonKeyFrames":false,"ManifestSubtitles":"vtt"},{"Container":"webm","Type":"Video","AudioCodec":"vorbis","VideoCodec":"vpx","Context":"Streaming","Protocol":"http","MaxAudioChannels":"2"},{"Container":"mp4","Type":"Video","AudioCodec":"mp3,aac,opus,flac,vorbis","VideoCodec":"h264","Context":"Static","Protocol":"http"}],"ContainerProfiles":[],"CodecProfiles":[{"Type":"VideoAudio","Codec":"aac","Conditions":[{"Condition":"Equals","Property":"IsSecondaryAudio","Value":"false","IsRequired":"false"}]},{"Type":"VideoAudio","Conditions":[{"Condition":"Equals","Property":"IsSecondaryAudio","Value":"false","IsRequired":"false"}]},{"Type":"Video","Codec":"h264","Conditions":[{"Condition":"EqualsAny","Property":"VideoProfile","Value":"high|main|baseline|constrained baseline|high 10","IsRequired":false},{"Condition":"LessThanEqual","Property":"VideoLevel","Value":"62","IsRequired":false},{"Condition":"LessThanEqual","Property":"Width","Value":"%d","IsRequired":false}]},{"Type":"Video","Codec":"hevc","Conditions":[{"Condition":"EqualsAny","Property":"VideoCodecTag","Value":"hvc1|hev1|hevc|hdmv","IsRequired":false},{"Condition":"LessThanEqual","Property":"Width","Value":"%d","IsRequired":false}]},{"Type":"Video","Conditions":[{"Condition":"LessThanEqual","Property":"Width","Value":"%d","IsRequired":false}]}],"SubtitleProfiles":[{"Format":"vtt","Method":"Hls"},{"Format":"eia_608","Method":"VideoSideData","Protocol":"hls"},{"Format":"eia_708","Method":"VideoSideData","Protocol":"hls"},{"Format":"vtt","Method":"External"},{"Format":"ass","Method":"External"},{"Format":"ssa","Method":"External"}],"ResponseProfiles":[{"Type":"Video","Container":"m4v","MimeType":"video/mp4"}]}}`

// 1920
// 1280

func getdeviceProfile(v int) string {
	if v == 0 {
		v = 3840
	}
	return fmt.Sprintf(defaultPlayBackProfile, v)
}
