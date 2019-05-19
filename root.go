package main

import "html/template"

var tmplRoot = template.Must(template.New("root").Parse(`
<html>
<head>
<title>Gopload</title>
</head>
<style>
progress {
  width: 100%;
}
td.status {
  width: 30%;
  text-align: right;
}
th.name {
  width: 30%;
}
#filetable {
  border: 1px solid black;
  margin-top: 1em;
  margin-bottom: 1em;
  width: 100%;
/*  display: none; */
}
#filetable th {
  border-bottom: 1px solid black;
}
</style>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/3.2.1/jquery.min.js"></script>
<script>
var chunk_size = 100000;

function status(n, s) {
  console.log("File " + n + ": " + s);
  $("#status-"+n).text(s);
}

function upload_part(f, ws, offset)
{
  var reader = new FileReader();
  var s = f.slice(offset, offset + chunk_size);
  reader.onload = function() {
    if (ws.readyState === ws.CLOSED) {
      return;
    }
    var data = reader.result;
    var size = data.byteLength;
    //console.log(data);
    //console.log("loaded part offset " + offset + " size: " + size);
    if (size > 0) {
      var sent = offset + size;
      // status("Sent " + sent + " of " + f.size + " " + (100.0*sent/f.size) + "%");
      ws.send(data);
      upload_part(f, ws, offset + size);
    } else {
      // status("All in flight...");
    }
  }
  reader.readAsArrayBuffer(s);
}

function upload_files(files, n) {
  if (files.length == n) {
    console.log("All uploads done");
    return;
  }
  var f = files[n];
  var error_set = false;
  console.log(f);
  ws = new WebSocket("wss://"+window.location.host+"{{.}}upload-ws/" + f.name);
  ws.onerror = function(err) {
    console.log("Websocket error: " + err);
  }
  ws.onopen = function() {
    upload_part(f, ws, 0);
  }
  ws.onerror = function(e) {
    console.log("Error: " + e);
  }
  ws.onclose = function(ev) {
    if (ev.code != 1000) {
      if (!error_set) {
        status(n, "Upload failed.");
      }
      console.log(ev);
    } else {
      status(n, "Uploaded " + f.size + " bytes");
      document.getElementById("progress-"+n).value = 100;
    }
    upload_files(files, n+1);
  }
  ws.onmessage = function(ev) {
    var o = JSON.parse(ev.data);
    if (o.Error !== undefined) {
      status(n, "Upload failed: " + o.Error);
      error_set = true;
    }
    if (o.Written !== undefined) {
      var p = 100.0*o.Written/f.size;
      status(n, "Sent " + o.Written + " of " + f.size + " " + Math.round(p) + "%");
      document.getElementById("progress-" + n).value = p;
      if (o.Written == f.size) {
        ws.close(1000)
      }
    }
  }
}

function upload(event)
{
  console.log("Uploading...");
  var fs = document.getElementById("files").files;
  var i;
  var o = $("#filetable");
  for (i = 0; i < fs.length; i++) {
    console.log("Adding file: " + fs[i].name);
    var $tr = $("<tr></tr>");

    $("<td></td>", {class:"name"}).text(fs[i].name).appendTo($tr);
    $("<td></td>", {id: "status-"+i, "class": "status"}).text("IDLE").appendTo($tr);
    $("<progress></progress>", {id: "progress-"+i, "max": "100", "value": "0"}).appendTo($("<td></td>").appendTo($tr));

    o.append($tr);
  }
  $("#filetable").css("display", "block");
  $("input,button").prop("disabled", true);
  upload_files(fs, 0);
}

</script>
<input id="files" type="file" name="file" accept="*/*" multiple />
<button onclick="upload()">Upload</button>
<table id="filetable">
  <tr>
    <th>File</th>
    <th>Status</th>
    <th>Progress</th>
  </tr>
</table>
</html>
`))
