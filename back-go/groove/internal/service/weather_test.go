package service

import "testing"

func TestClassifyWeather(t *testing.T) {
	cases := []struct {
		code     int
		category string
	}{
		{0, "clear"},
		{2, "clouds"},
		{3, "overcast"},
		{45, "fog"},
		{53, "drizzle"},
		{61, "rain"},
		{82, "rain"}, // ливень — тоже дождь
		{71, "snow"},
		{86, "snow"},
		{95, "storm"},
		{99, "storm"},
		{42, "clouds"}, // неизвестный код → нейтральная облачность
	}
	for _, c := range cases {
		if cat, _, _ := classifyWeather(c.code); cat != c.category {
			t.Errorf("code %d: категория %q, ожидалась %q", c.code, cat, c.category)
		}
	}
}

func snap(category string, temp float64) weatherSnapshot {
	return weatherSnapshot{Category: category, TempC: temp}
}

func TestWeatherTransition(t *testing.T) {
	cases := []struct {
		name string
		prev weatherSnapshot
		cur  weatherSnapshot
		want string
	}{
		{"начался дождь", snap("clouds", 10), snap("rain", 9), "rain"},
		{"морось→дождь не считается", snap("drizzle", 10), snap("rain", 10), ""},
		{"пошёл снег", snap("overcast", -2), snap("snow", -3), "snow"},
		{"гроза", snap("rain", 18), snap("storm", 17), "storm"},
		{"туман", snap("clear", 5), snap("fog", 5), "fog"},
		{"прояснилось после дождя", snap("rain", 12), snap("clear", 14), "cleared"},
		{"дождь→пасмурно — молчим", snap("rain", 12), snap("overcast", 12), ""},
		{"туман→ясно — молчим", snap("fog", 5), snap("clear", 6), ""},
		{"облачность меняется — молчим", snap("clear", 20), snap("clouds", 20), ""},
		{"жара началась", snap("clear", 28), snap("clear", 31), "heat"},
		{"жара продолжается — молчим", snap("clear", 32), snap("clear", 33), ""},
		{"мороз ударил", snap("overcast", -10), snap("overcast", -16), "frost"},
		{"без перемен", snap("rain", 10), snap("rain", 11), ""},
	}
	for _, c := range cases {
		if got := weatherTransition(c.prev, c.cur); got != c.want {
			t.Errorf("%s: %q, ожидалось %q", c.name, got, c.want)
		}
	}
}

func TestFormatTempC(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{12.4, "+12°C"},
		{-3.6, "-4°C"},
		{0.2, "0°C"},
		{29.6, "+30°C"},
	}
	for _, c := range cases {
		if got := formatTempC(c.in); got != c.want {
			t.Errorf("formatTempC(%v) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}
