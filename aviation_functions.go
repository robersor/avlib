package avlib

import (
	"math"
)

func EstimatedTimeEnrouteMin(distance float64, ground_speed_kts float64) float64 {
	return distance / ground_speed_kts * 60
}

func CalculateGroundSpeed(wind_direction_deg float64, course_deg float64, true_airspeed_kts float64, wind_speed_kts float64) float64 {
	wdir_rad := DegreeToRadian(wind_direction_deg)
	course_rad := DegreeToRadian(course_deg)
	wca_deg := CalculateWindCorrectionAngle(wind_direction_deg, course_deg, true_airspeed_kts, wind_speed_kts)
	wca_rad := DegreeToRadian(wca_deg)
	tas_sq := true_airspeed_kts * true_airspeed_kts
	ws_sq := wind_speed_kts * wind_speed_kts
	return math.Sqrt(tas_sq + ws_sq - (2 * true_airspeed_kts * wind_speed_kts * math.Cos(course_rad-wdir_rad+wca_rad)))
}

func CalculateWindCorrectionAngle(wind_direction_deg float64, course_deg float64, true_airspeed_kts float64, wind_speed_kts float64) float64 {
	wdir_rad := DegreeToRadian(wind_direction_deg)
	course_rad := DegreeToRadian(course_deg)
	wind_x_course_ang := wdir_rad - course_rad
	speed_ratio := wind_speed_kts / true_airspeed_kts
	wind_corr_ang_rad := math.Asin(speed_ratio * math.Sin(wind_x_course_ang))
	return RadianToDegree(wind_corr_ang_rad)
}

func GetCrosswind(runway_heading float64, wind_heading float64, wind_speed_kts float64) float64 {
	// wind to runway angle
	diff := math.Abs(runway_heading - wind_heading)
	if diff > 180 {
		diff = 360 - diff
	}
	ang_diff_rad := DegreeToRadian(diff)

	direct_component := wind_speed_kts * math.Sin(ang_diff_rad)

	if diff > 90 {
		return -1 * direct_component
	}
	return direct_component

}

func GetHeadWindComponent(runway_heading float64, wind_heading float64, wind_speed_kts float64) float64 {
	// wind to runway angle
	diff := math.Abs(runway_heading - wind_heading)
	if diff > 180 {
		diff = 360 - diff
	}
	ang_diff_rad := DegreeToRadian(diff)

	direct_component := wind_speed_kts * math.Cos(ang_diff_rad)

	if diff > 90 {
		return -1 * direct_component
	}
	return direct_component

}

func GetFuelBurned(burnrate float64, ete_minutes float64) float64 {
	var fb float64
	fb = burnrate * ete_minutes / 60
	fb = math.Ceil(fb*10) / 10
	return fb
}

func DegreeToRadian(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func RadianToDegree(radian float64) float64 {
	return radian * 180 / math.Pi
}
