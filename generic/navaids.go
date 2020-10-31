package generic

import (
	"log"

	"github.com/PuerkitoBio/goquery"
)

// Navaids descvribes the navigation means available on the airport/
type Navaid struct {
	Id              string
	Name            string
	Frequency       string
	NavaidType      string
	MagVar          string
	OperationsHours string
	Position        GeoPosition
	Elevation       string
	Remarks         string
	Key             string
}

func (n *Navaid) SetFromHtmlSelection(tr *goquery.Selection) {
	tr.Find("td").Each(func(index int, td *goquery.Selection) {
		switch index {
		case 0:
			n.setColumn0(td)
		case 1:
			n.Id = td.Text()
		case 2:
			n.Frequency = td.Text()
		case 3:
			n.OperationsHours = td.Text()
		case 4:
			//n.position = td.Text()
			n.setColumn4(td)
		case 5:
			n.Elevation = td.Text()
		case 6:
			n.Remarks = td.Text()
		}
		n.Key = n.Id + " " + n.NavaidType
	})
}

func (n *Navaid) setColumn0(html *goquery.Selection) {
	var data []string
	fs := html.Text()
	html.Find("p").Each(func(index int, shtml *goquery.Selection) {
		data = append(data, shtml.Text())
	})

	switch len(data) {
	case 1:
		n.NavaidType = data[0]
		n.Name = fs[0 : len(fs)-len(data[0])]
	case 2:
		n.NavaidType = data[0]
		n.MagVar = data[1]
		n.Name = fs[0 : len(fs)-len(data[0])-len(data[1])]
	}
}

func (n *Navaid) setColumn4(html *goquery.Selection) {
	var data []string
	html.Find("p").Each(func(index int, shtml *goquery.Selection) {
		data = append(data, shtml.Text())
	})

	if len(data) == 2 {
		lat, err := ConvertDDMMSSSSLatitudeToFloat(data[0])
		if err != nil {
			log.Printf("%s Latitude Conversion problem %s \n", n.Name, data[0])
			log.Println(err)
		} else {
			n.Position.Latitude = lat
		}

		long, err := ConvertDDDMMSSSSLongitudeToFloat(data[1])
		if err != nil {
			log.Printf("%s Longitude Conversion problem %s \n", n.Name, data[1])
			log.Println(err)
		} else {
			n.Position.Longitude = long
		}
	} else {
		log.Printf("%s Conversion problem %s \n", n.Name)
	}
}

func (n *Navaid) CompareTo(ext *Navaid) bool {
	if n.Key == ext.Key {
		return true
	} else {
		if (n.Id == ext.Id) && (n.NavaidType == ext.NavaidType) {
			return true
		} else {
			return false
		}
	}
}

func (n *Navaid) IsInMap(m *map[string]Navaid) bool {
	for _, in := range *m {
		if n.CompareTo(&in) {
			return true
		}
	}
	return false
}
