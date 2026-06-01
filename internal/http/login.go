package http

import (
	"net/http"
)

type loginData struct {
	Error string
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if s.auth.IsAuthenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render.Render(w, r, "login.html", loginData{})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Login(w, r); err != nil {
		s.render.Render(w, r, "login.html", loginData{
			Error: "Invalid username or password",
		})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.auth.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
