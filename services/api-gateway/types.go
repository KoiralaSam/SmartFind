package main

type PassengerLoginRequest struct {
	IDToken string `json:"id_token"`
}

type PassengerDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
}

type PassengerLoginResponse struct {
	Passenger    PassengerDTO `json:"passenger"`
	SessionToken string       `json:"session_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
