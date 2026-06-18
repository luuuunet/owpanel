package analytics

// echartsMapName maps ISO codes to names used in ECharts world.json.
func echartsMapName(code, fallback string) string {
	if n, ok := echartsCountryNames[code]; ok {
		return n
	}
	if fallback != "" {
		return fallback
	}
	return code
}

var echartsCountryNames = map[string]string{
	"CN": "China",
	"US": "United States",
	"GB": "United Kingdom",
	"RU": "Russia",
	"KR": "South Korea",
	"KP": "North Korea",
	"TW": "Taiwan",
	"HK": "Hong Kong",
	"MO": "Macao",
	"VN": "Vietnam",
	"IN": "India",
	"BR": "Brazil",
	"AU": "Australia",
	"CA": "Canada",
	"DE": "Germany",
	"FR": "France",
	"IT": "Italy",
	"ES": "Spain",
	"NL": "Netherlands",
	"JP": "Japan",
	"SG": "Singapore",
	"MY": "Malaysia",
	"TH": "Thailand",
	"PH": "Philippines",
	"ID": "Indonesia",
	"MX": "Mexico",
	"AR": "Argentina",
	"ZA": "South Africa",
	"EG": "Egypt",
	"TR": "Turkey",
	"SA": "Saudi Arabia",
	"AE": "United Arab Emirates",
	"IL": "Israel",
	"IR": "Iran",
	"IQ": "Iraq",
	"PK": "Pakistan",
	"BD": "Bangladesh",
	"NZ": "New Zealand",
	"PL": "Poland",
	"UA": "Ukraine",
	"SE": "Sweden",
	"NO": "Norway",
	"DK": "Denmark",
	"FI": "Finland",
	"CH": "Switzerland",
	"AT": "Austria",
	"BE": "Belgium",
	"PT": "Portugal",
	"GR": "Greece",
	"CZ": "Czech Rep.",
	"HU": "Hungary",
	"RO": "Romania",
	"IE": "Ireland",
}

func countryCentroid(code string) (lat, lng float64) {
	if c, ok := countryCentroids[code]; ok {
		return c[0], c[1]
	}
	return 0, 0
}

var countryCentroids = map[string][2]float64{
	"CN": {35.8617, 104.1954}, "US": {37.0902, -95.7129}, "GB": {55.3781, -3.4360},
	"JP": {36.2048, 138.2529}, "DE": {51.1657, 10.4515}, "FR": {46.2276, 2.2137},
	"IN": {20.5937, 78.9629}, "BR": {-14.2350, -51.9253}, "RU": {61.5240, 105.3188},
	"KR": {35.9078, 127.7669}, "SG": {1.3521, 103.8198}, "AU": {-25.2744, 133.7751},
	"CA": {56.1304, -106.3468}, "IT": {41.8719, 12.5674}, "ES": {40.4637, -3.7492},
	"NL": {52.1326, 5.2913}, "HK": {22.3193, 114.1694}, "TW": {23.6978, 120.9605},
	"MX": {23.6345, -102.5528}, "AR": {-38.4161, -63.6167}, "ZA": {-30.5595, 22.9375},
	"TR": {38.9637, 35.2433}, "SA": {23.8859, 45.0792}, "AE": {23.4241, 53.8478},
	"ID": {-0.7893, 113.9213}, "TH": {15.8700, 100.9925}, "VN": {14.0583, 108.2772},
	"PH": {12.8797, 121.7740}, "MY": {4.2105, 101.9758}, "PL": {51.9194, 19.1451},
	"UA": {48.3794, 31.1656}, "SE": {60.1282, 18.6435}, "CH": {46.8182, 8.2275},
	"NO": {60.4720, 8.4689}, "DK": {56.2639, 9.5018}, "FI": {61.9241, 25.7482},
	"BE": {50.5039, 4.4699}, "AT": {47.5162, 14.5501}, "PT": {39.3999, -8.2245},
	"GR": {39.0742, 21.8243}, "IE": {53.4129, -8.2439}, "NZ": {-40.9006, 174.8860},
	"EG": {26.8206, 30.8025}, "NG": {9.0820, 8.6753}, "PK": {30.3753, 69.3451},
	"BD": {23.6849, 90.3563}, "IL": {31.0461, 34.8516}, "IR": {32.4279, 53.6880},
	"IQ": {33.2232, 43.6793}, "CL": {-35.6751, -71.5430}, "CO": {4.5709, -74.2973},
	"PE": {-9.1900, -75.0152}, "VE": {6.4238, -66.5897}, "KZ": {48.0196, 66.9237},
}

func cityCoord(city string) (lat, lng float64, ok bool) {
	c, exists := cityCoords[city]
	if !exists {
		return 0, 0, false
	}
	return c[0], c[1], true
}

var cityCoords = map[string][2]float64{
	"Shanghai": {31.2304, 121.4737}, "Beijing": {39.9042, 116.4074},
	"New York": {40.7128, -74.0060}, "Los Angeles": {34.0522, -118.2437},
	"Tokyo": {35.6762, 139.6503}, "Singapore": {1.3521, 103.8198},
	"Frankfurt": {50.1109, 8.6821}, "London": {51.5074, -0.1278},
	"Seoul": {37.5665, 126.9780}, "Paris": {48.8566, 2.3522},
	"Mumbai": {19.0760, 72.8777}, "Sydney": {-33.8688, 151.2093},
	"São Paulo": {-23.5505, -46.6333}, "Moscow": {55.7558, 37.6173},
	"Toronto": {43.6532, -79.3832}, "Amsterdam": {52.3676, 4.9041},
	"Hong Kong": {22.3193, 114.1694}, "Taipei": {25.0330, 121.5654},
}
