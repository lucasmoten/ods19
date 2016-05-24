
var __state = {};

var BASE_SERVICE_URL = '/services/object-drive/1.0/'


function getCN(dn) {
  return dn.substring(dn.indexOf('=')+1, dn.indexOf(','))
}

function newParent(id) {
  __state.parentId = id;
  refreshListObjects();
};

function listUsers() {
  return $.ajax({
    url: BASE_SERVICE_URL+'users',
    contentType: 'application/json',
    method: 'GET'
  });
};

function refreshListObjects() {
  var url;
  var t = $('#listObjectResults');
  // remove children first
  $('#listObjectResults tbody > tr').remove();
  // choose correct listObjects URL
  if (__state.parentId === "") {
    url  = BASE_SERVICE_URL + 'objects';
  } else {
    url = BASE_SERVICE_URL + 'objects/' + __state.parentId
  }
  // paging - eventually capture in __state?
  url += '?pageNumber=1&pageSize=20'
  reqwest({
      url: url
    , method: 'get'
    , type: 'json'
    , contentType: 'application/json'
   // , data: JSON.stringify({ pageNumber: 1, pageSize: 20, parentId: __state.parentId })
    , success: function (resp) {
      $.when(listUsers()).done(function (userdata) {

        __state.users = userdata[0];
        __state.Objects = resp.objects;

        $.each(resp.objects, function(index, item){
          $('#listObjectResults').append(_renderListObjectRow(index, item));
        })

        }).then(function(){
        // set up share handlers
          for ( var i = 0; i < resp.objects.length; i++ ) {
            (function (_rowId, _obj) {
              $(_rowId).siblings('.shareButton').click(function() {
                var dn = $(_rowId).val();
                doShare(_obj.id, dn)
              })
            })('#listObjectRow_' + i, __state.Objects[i])
          }
          for ( var i = 0; i < resp.objects.length; i++ ) {
            (function (_drowId, _obj) {
              $(_drowId).click(function() {
                doDelete(_obj.id, _obj.changeToken)
              })
            })('#deleteObjectRow_' + i, __state.Objects[i])
          }
      });
    }
    , error: function(resp) {
      console.log("refresh list objects failed!")
    }
  })
};

function refreshSharedWithMe() {
  $('#listSharedWithMeResults tbody > tr').remove();
  $.ajax({
    url: BASE_SERVICE_URL + 'shares',
    contentType: 'application/json',
    method: 'GET',
    success: function(resp) {
      _renderSharedWithMeTable(resp.objects);
    },
    error: function(resp) {
      console.log("refresh shared with me failed!")
    }
  });
};


function _renderSharedWithMeTable(objs) {
  $.each(objs, function(index, item){
    $('#listSharedWithMeResults').append(_renderSharedWithMeRow(index, item));
  })
}

function _renderSharedWithMeRow(index, item) {

  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + getCN(item.createdBy) + '</td>';
  var size = '<td align=right>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';

  return '<tr>' +
     name + type + createdDate + createdBy + size + changeToken + acm +
     '</tr>';
}

function doShare(objectId, userId, opts) {
  if (!opts) {
    opts = { create: true, read: true, update: false, delete: false, share: false, propogateToChildren: true}
  }
  var data = {
    grantee: userId,
    create: opts.create,
    read: opts.read,
    update: opts.update,
    delete: opts.delete,
    share: opts.share,
    propogateToChildren: opts.propogateToChildren
  };
  $.ajax({
    url: BASE_SERVICE_URL + 'shared/' + objectId,
    contentType: 'application/json',
    method: 'POST',
    data: JSON.stringify(data),
    success: function(resp) {
      refreshSharedWithMe();
    },
    error: function(resp) {
      console.log("do share failed!");
      console.log(resp);  
    }
  });
};

function doDelete(objectId, changeToken) {
  var folderName = $("#folderNameInput").val();
  var data = {
    changeToken: changeToken
  }
  $.ajax({
      url: BASE_SERVICE_URL+'objects/'+objectId+'/trash',
      method: 'POST',
      data: JSON.stringify(data),
      contentType: 'application/json',
      success: function(data){
        refreshListObjects();
      },
      error: function(data) {
        console.log("do delete failed!")  
      }
  });
}


// Return a <tr> string suitable to append to table.
function _renderListObjectRow(index, item, elm) {

  var rowId = 'listObjectRow_' + index;
  var drowId = 'deleteObjectRow_' + index;

  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + getCN(item.createdBy) + '</td>';
  var size = '<td align=right>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';
  var hash = '<td>' + item.hash + '</td>';
  var shareDropdown = _renderUsersDropdown(item, __state.users, rowId);
  var deleteButton = _renderDeleteButton(item, __state.users, drowId);

  return '<tr>' +
     name + type + createdDate + createdBy + size + changeToken + acm + shareDropdown + deleteButton +
     '</tr>';
}

function _renderUsersDropdown(obj, users, rowId) {
  var sel = $('<select></select>');
   sel.append($("<option>").attr('value', '').text('--'));
  for ( i=0; i < users.length; i ++ ) {
   sel.append($("<option>").attr('value', users[i].distinguishedName).text(users[i].displayName));
  }
  return '<td><select id="' + rowId + '">' + sel.html() + '</select><button class="shareButton">share</button></td>'
};

function _renderDeleteButton(obj, users, drowId) {
  return '<td><button id="'+drowId+'" class="deleteButton">delete</button></td>'
}

// Render a proper href, depending on whether an object is a folder or an object proper.
function _renderObjectLink(item) {
  var link;
  if (item.typeName === "Folder") {
    link = '<td><a href="'+ BASE_SERVICE_URL + 'ui/listObjects?parentId=' + item.id + '">'+item.name+'</a></td>';
  } else {
    link = '<td><a href="'+ BASE_SERVICE_URL + 'objects/' + item.id + '/stream">' + item.name + '</a></td>';
  }
  return link;
}

function createObject() {
      console.log("createObject called");
      // get the form data
      var objectName = $("#newObjectName").val();
      //var classification = $("#classification").val();
      var jsFileObject = $("#fileHandle")[0].files[0];
      var contentType = jsFileObject.type || "text/plain";
      var fileName = jsFileObject.name;
      var size = jsFileObject.size;
      //var rawAcm = '{"version":"2.1.0","classif":"'+classification+'"}'
      var rawAcm = '{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[\"FOUO\"],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U//FOUO\",\"banner\":\"UNCLASSIFIED//FOUO\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}'
      
      var req = {
        acm: rawAcm,
        title: objectName,
        fileName: fileName,
        size: size,
        contentType: contentType,
        parentId: __state.parentId
      }

      // call the server with the data
      var formData = new FormData();
      formData.append("ObjectMetadata", JSON.stringify(req));
      formData.append("filestream", jsFileObject);

      $("#submitCreateObject").prop("disabled", true)
      $.ajax({
        url: BASE_SERVICE_URL+'objects',
        data: formData,
        cache: false,
        contentType: false,
        processData: false,
        method: 'POST',
        success: function(data){
          refreshListObjects();
          $("#submitCreateObject").prop("disabled", false)
        },
        error: function(data) {
          console.log("create object failed!")  
          $("#submitCreateObject").prop("disabled", false)
        }
      });
}

function createFolder() {
  var folderName = $("#folderNameInput").val();
  
  var rawAcm = '{\"version\":\"2.1.0\",\"classif\":\"U\",\"owner_prod\":[\"USA\"],\"atom_energy\":[],\"sar_id\":[],\"sci_ctrls\":[],\"disponly_to\":[\"\"],\"dissem_ctrls\":[],\"non_ic\":[],\"rel_to\":[],\"fgi_open\":[],\"fgi_protect\":[],\"portion\":\"U\",\"banner\":\"UNCLASSIFIED\",\"dissem_countries\":[\"USA\"],\"accms\":[],\"macs\":[],\"oc_attribs\":[{\"orgs\":[],\"missions\":[],\"regions\":[]}],\"f_clearance\":[\"u\"],\"f_sci_ctrls\":[],\"f_accms\":[],\"f_oc_org\":[],\"f_regions\":[],\"f_missions\":[],\"f_share\":[],\"f_atom_energy\":[],\"f_macs\":[],\"disp_only\":\"\"}'
  
  
  var data = {
    typeName: "Folder",
    parentId: __state.parentId,
    name: folderName,
    acm: rawAcm,
  }
      $.ajax({
      url: BASE_SERVICE_URL+'objects',
      data: JSON.stringify(data),
      method: 'POST',
      contentType: 'application/json',
      success: function(data){
        console.log("createFolder success.")
        console.log(data);
        refreshListObjects();
      },
      error: function(data) {
        console.log("createFolder failed!")
      }
  });
}

function shareTo() {};

function init() {
  // Set up click handlers
  $("#submitCreateObject").click(createObject);
  $("#refreshListObjects").click(refreshListObjects);
  $("#submitCreateFolder").click(createFolder);
  $("#refreshSharedWithMe").click(refreshSharedWithMe);
  $("#refreshListObjectsShared").click(refreshObjectsIShared);

  // Get parentId from hidden field, if set.
  __state.parentId = $('#hiddenParentId').attr('data-value') || "";
  __state.users = [];

  refreshListObjects();
  refreshSharedWithMe();
  refreshObjectsIShared();
};

function refreshObjectsIShared() {
  $('#listUserObjectsSharedResults tbody > tr').remove();
  $.ajax({
    url: BASE_SERVICE_URL + 'shared',
    contentType: 'application/json',
    method: 'GET',
    success: function(resp) {
      _renderObjectsISharedTable(resp.objects);
    },
    error: function(resp) {
      console.log("refresh objects i shared failed!")
    }
  });
};


function _renderObjectsISharedTable(objs) {
  $.each(objs, function(index, item){
    $('#listUserObjectsSharedResults').append(_renderObjectsISharedRow(index, item));
  })
}

function _renderObjectsISharedRow(index, item) {

  var name = _renderObjectLink(item);
  var type = '<td>' + item.contentType + '</td>';
  var createdDate = '<td>' + item.createdDate + '</td>';
  var createdBy = '<td>' + getCN(item.createdBy) + '</td>';
  var size = '<td align=right>' + item.contentSize + '</td>';
  var changeToken = '<td>' + item.changeToken + '</td>';
  var acm = '<td>' + item.acm + '</td>';

  return '<tr>' +
     name + type + createdDate + createdBy + size + changeToken + acm +
     '</tr>';
}

$(document).ready(init);
