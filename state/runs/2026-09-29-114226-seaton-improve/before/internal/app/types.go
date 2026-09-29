package app

import "time"

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	Email       string     `json:"email,omitempty"`
	EmployeeID  *string    `json:"employeeId,omitempty"`
	Role        string     `json:"role"`
	Source      string     `json:"source"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	Active      bool       `json:"active"`
}

func (u User) IsAdmin() bool { return u.Role == "system_admin" }
func (u User) CanManageSeats() bool {
	return u.Role == "system_admin" || u.Role == "seat_manager"
}

type Setting struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Secret     bool   `json:"secret"`
	Configured bool   `json:"configured"`
}

type Employee struct {
	ID               string  `json:"id"`
	EmployeeNo       string  `json:"employeeNo"`
	Name             string  `json:"name"`
	Email            string  `json:"email,omitempty"`
	OrganizationID   *string `json:"organizationId,omitempty"`
	OrganizationName string  `json:"organizationName,omitempty"`
	Title            string  `json:"title,omitempty"`
	Position         string  `json:"position,omitempty"`
	Workplace        string  `json:"workplace,omitempty"`
	Status           string  `json:"status"`
	SeatID           *string `json:"seatId,omitempty"`
	SeatNo           string  `json:"seatNo,omitempty"`
}

type Seat struct {
	ID               string   `json:"id"`
	FloorMapID       string   `json:"floorMapId"`
	SeatNo           string   `json:"seatNo"`
	Type             string   `json:"type"`
	Status           string   `json:"status"`
	X                float64  `json:"x"`
	Y                float64  `json:"y"`
	Width            float64  `json:"width"`
	Height           float64  `json:"height"`
	Rotation         float64  `json:"rotation"`
	Confidence       *float64 `json:"confidence,omitempty"`
	OrganizationID   *string  `json:"organizationId,omitempty"`
	OrganizationName string   `json:"organizationName,omitempty"`
	EmployeeID       *string  `json:"employeeId,omitempty"`
	EmployeeNo       string   `json:"employeeNo,omitempty"`
	EmployeeName     string   `json:"employeeName,omitempty"`
	// 좌석맵에서 조직별 색상과 구역 불일치를 표시하려면 배정된 직원의 조직도 필요하다.
	// OrganizationID 는 좌석에 지정된 구역, 아래는 실제로 앉은 직원의 소속이다.
	EmployeeOrganizationID   *string `json:"employeeOrganizationId,omitempty"`
	EmployeeOrganizationName string  `json:"employeeOrganizationName,omitempty"`
	// 좌석 상세는 직원 검색 결과가 아니라 좌석 응답만으로 채워져야 한다.
	// 검색하지 않고 좌석을 고른 경우에도 근무지를 보여주기 위한 값이다.
	EmployeeWorkplace string `json:"employeeWorkplace,omitempty"`
}

type ctxKey string

const userContextKey ctxKey = "user"
const csrfContextKey ctxKey = "csrf"
const apiScopesContextKey ctxKey = "api_scopes"
