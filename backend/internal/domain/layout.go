package domain

type Site struct {
	Center   Point
	RadiusM  float64
	Capacity *int64
}

func CheckBinLayout(site Site, position Point, capacity *int64) map[string]string {
	fields := map[string]string{}
	if Distance(position, site.Center) > site.RadiusM {
		fields["x"] = "bin must be within the warehouse perimeter"
	}
	if capacity != nil {
		switch {
		case *capacity <= 0:
			fields["capacity"] = "must be greater than zero, or left empty for no limit"
		case site.Capacity != nil && *capacity > *site.Capacity:
			fields["capacity"] = "must not exceed the warehouse capacity"
		}
	}
	return fields
}
