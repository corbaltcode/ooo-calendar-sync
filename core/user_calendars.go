package core

type UserCalendarMapping struct {
	UserEmail string
	Calendars []CalendarTarget
}

type CalendarTarget struct {
	CalendarID   string
	CalendarName string
}
