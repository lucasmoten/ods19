
// TODO change this after we merge in Rob's request object code
function doRequest() {
  $.ajax({
    url: "test.html",
    context: document.body
  }).done(function() {
    $( this ).addClass( "done" );
  });
};
