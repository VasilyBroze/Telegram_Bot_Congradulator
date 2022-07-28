package entity

//НАСТРОЙКИ БОТА
type BotSettings struct {
	Google_sheet_bday_url  string `json:"google_bday_url"`
	Google_sheet_bday_list string `json:"google_bday_list"`
	Google_sheet_text_url  string `json:"google_text_url"`
	Google_sheet_text_list string `json:"google_text_list"`
	Bot_token              string `json:"bot_token"`
	Chat_id                int64  `json:"chat_id"`
	Anonce_pass            string `json:"anonce_password"`
}
