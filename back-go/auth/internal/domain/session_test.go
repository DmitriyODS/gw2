package domain

import "testing"

func TestDescribeUserAgent(t *testing.T) {
	cases := []struct {
		name                     string
		ua                       string
		platform, client, device string
	}{
		{
			name:     "Capacitor-обёртка: модель телефона из UA",
			ua:       "Mozilla/5.0 (Linux; Android 14; Pixel 8 Build/UP1A.231005.007; wv) AppleWebKit/537.36 Chrome/120.0 Mobile Safari/537.36 GrooveWorkApp",
			platform: PlatformMobile, client: ClientApp, device: "Pixel 8",
		},
		{
			name:     "обёртка без узнаваемой модели",
			ua:       "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 GrooveWorkApp",
			platform: PlatformMobile, client: ClientApp, device: "Android",
		},
		{
			name:     "Electron: остаётся ОС",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 GrooveWork/1.0.6 Chrome/126.0 Electron/30.0.0 Safari/537.36",
			platform: PlatformDesktop, client: ClientApp, device: "MAC OS",
		},
		{
			name:     "обычный Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36",
			platform: PlatformWeb, client: ClientWeb, device: "Chrome · Windows",
		},
		{
			// Edge и Яндекс.Браузер представляются и как Chrome, и как Safari:
			// специфичная метка обязана выигрывать.
			name:     "Edge не выдаёт себя за Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0 Safari/537.36 Edg/126.0",
			platform: PlatformWeb, client: ClientWeb, device: "Edge · Windows",
		},
		{
			name:     "Яндекс.Браузер",
			ua:       "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/124.0 YaBrowser/24.6.0 Safari/537.36",
			platform: PlatformWeb, client: ClientWeb, device: "Яндекс.Браузер · Windows",
		},
		{
			name:     "мобильный браузер: вместо ОС модель",
			ua:       "Mozilla/5.0 (Linux; Android 14; SM-S911B) AppleWebKit/537.36 Chrome/126.0 Mobile Safari/537.36",
			platform: PlatformWeb, client: ClientWeb, device: "Chrome · SM-S911B",
		},
		{
			name:     "пустой UA (серверный запрос)",
			ua:       "",
			platform: PlatformWeb, client: ClientWeb, device: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			platform, client, device := DescribeUserAgent(c.ua)
			if platform != c.platform || client != c.client || device != c.device {
				t.Fatalf("got (%s, %s, %q), want (%s, %s, %q)",
					platform, client, device, c.platform, c.client, c.device)
			}
		})
	}
}
