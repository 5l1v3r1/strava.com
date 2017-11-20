var initMap = function() {
  var map = new google.maps.Map(document.getElementById('map'), {mapTypeId: 'satellite'});
  bounds  = new google.maps.LatLngBounds();
  routes.map(function(route) {
    var path = route.map(function(point) {
      bounds.extend(new google.maps.LatLng(point[0], point[1]));
      return {
        lat: point[0],
        lng: point[1],
      }
    });
    var polyline = new google.maps.Polyline({
      geodesic: true,
      path: path,
      strokeColor: '#ff0000',
      strokeOpacity: 1.0,
      strokeWeight: 2,
    });
    polyline.setMap(map);
  });
  map.fitBounds(bounds);
  map.panToBounds(bounds);
}
