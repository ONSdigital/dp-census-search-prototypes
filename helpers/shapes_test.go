package helpers_test

import (
	"testing"

	"github.com/ONSdigital/dp-census-search-prototypes/helpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCircleToPolygon(t *testing.T) {
	Convey("Given the number of segments does not exceed 180", t, func() {
		geopoint := helpers.Coordinate{
			Lat: 51.486090,
			Lon: -3.227882,
		}

		expectedCoordinates := []helpers.Coordinate{
			{Lat: 51.495073152841194, Lon: -3.2278819999999997},
			{Lat: 51.494633401528986, Lon: -3.232340722565203},
			{Lat: 51.49335721753477, Lon: -3.2363627568939477},
			{Lat: 51.49136958551781, Lon: -3.239554251574248},
			{Lat: 51.488865146452405, Lon: -3.2416028009712456},
			{Lat: 51.48608911512211, Lon: -3.24230802558419},
			{Lat: 51.48331325278844, Lon: -3.241601130457992},
			{Lat: 51.48080925616197, Lon: -3.2395515486269555},
			{Lat: 51.47882217102961, Lon: -3.236360053946569},
			{Lat: 51.47754642947436, Lon: -3.2323390520518105},
			{Lat: 51.4771068471588, Lon: -3.2278819999999997},
			{Lat: 51.47754642947436, Lon: -3.223424947948189},
			{Lat: 51.47882217102961, Lon: -3.2194039460534305},
			{Lat: 51.48080925616197, Lon: -3.216212451373044},
			{Lat: 51.48331325278844, Lon: -3.2141628695420073},
			{Lat: 51.48608911512211, Lon: -3.2134559744158095},
			{Lat: 51.488865146452405, Lon: -3.2141611990287533},
			{Lat: 51.49136958551781, Lon: -3.2162097484257512},
			{Lat: 51.49335721753477, Lon: -3.219401243106052},
			{Lat: 51.494633401528986, Lon: -3.223423277434797},
			{Lat: 51.495073152841194, Lon: -3.2278819999999997},
		}

		shape, err := helpers.CircleToPolygon(geopoint, 1000, 20)
		So(err, ShouldBeNil)
		So(shape, ShouldNotBeNil)
		So(shape.Type, ShouldEqual, "Polygon")
		So(len(shape.Coordinates), ShouldEqual, 21)
		So(shape.Coordinates, ShouldResemble, expectedCoordinates)
	})

	Convey("Given the number of segments does exceed 180", t, func() {
		geopoint := helpers.Coordinate{
			Lat: 51.486090,
			Lon: -3.227882,
		}

		shape, err := helpers.CircleToPolygon(geopoint, 1000, 181)
		So(shape, ShouldBeNil)
		So(err, ShouldResemble, helpers.ErrTooManySegments)
	})

	Convey("Given the coordinates of centre point are incorrect", t, func() {

		Convey("When the lattitudinal coordinate of centre point is greater than 90", func() {
			geopoint := helpers.Coordinate{
				Lat: 90.1,
				Lon: -3.227882,
			}

			shape, err := helpers.CircleToPolygon(geopoint, 1000, 10)
			So(shape, ShouldBeNil)
			So(err, ShouldResemble, helpers.ErrInvalidLatitudinalPoint)
		})

		Convey("When the lattitudinal coordinate of centre point is less than -90", func() {
			geopoint := helpers.Coordinate{
				Lat: -90.1,
				Lon: -3.227882,
			}

			shape, err := helpers.CircleToPolygon(geopoint, 1000, 10)
			So(shape, ShouldBeNil)
			So(err, ShouldResemble, helpers.ErrInvalidLatitudinalPoint)
		})

		Convey("When the longitudinal coordinate of centre point is greater than 180", func() {
			geopoint := helpers.Coordinate{
				Lat: 51.486090,
				Lon: 180.227882,
			}

			shape, err := helpers.CircleToPolygon(geopoint, 1000, 10)
			So(shape, ShouldBeNil)
			So(err, ShouldResemble, helpers.ErrInvalidLongitudinalPoint)
		})

		Convey("When the longitudinal coordinate of centre point is less than -180", func() {
			geopoint := helpers.Coordinate{
				Lat: 51.486090,
				Lon: -180.227882,
			}

			shape, err := helpers.CircleToPolygon(geopoint, 1000, 10)
			So(shape, ShouldBeNil)
			So(err, ShouldResemble, helpers.ErrInvalidLongitudinalPoint)
		})
	})
}
