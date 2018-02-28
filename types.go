package main

type File struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type LanguagePostData struct {
	Cmd   string `json:"cmd"`
	Files []File `json:"files"`
}

type LanguageGetData struct {
	Language   string
	Version    string
}

type IndexGetData struct {
}
