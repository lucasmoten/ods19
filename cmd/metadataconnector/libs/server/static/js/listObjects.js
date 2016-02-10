
// TODO change this after we merge in Rob's request object code
function doRequest() {

  var t = $('#listObjectsResults tr:last');

  reqwest({
      url: '/services/metadataconnector/1.0/objects'
    , method: 'post'
    , type: 'json'
    , contentType: 'application/json'
    , data: { pageNumber: '1', pageSize: 20 }
    , success: function (resp) {
        // qwery('#listObjectsResults tr:last').html(resp)
        console.log(resp)
      }
  })
};

document.onload = doRequest();
