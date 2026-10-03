// Shared by the build (tests, no-JS default hrefs) and the page script.
// The build inlines this file into the page with the `export` keywords removed,
// so keep it plain ES5-compatible function syntax with no imports.

var EDITORS = [
  { id: "vscode-wsl", label: "VS Code (WSL)" },
  { id: "vscode-win", label: "VS Code (Windows)" },
  { id: "zed", label: "Zed" },
  { id: "cursor-wsl", label: "Cursor (WSL)" },
  { id: "copy", label: "Copy path" },
];

var DEFAULT_EDITOR = "vscode-wsl";

// /mnt/c/git/x -> C:/git/x. Other paths are returned unchanged.
function winPath(absPath) {
  var m = /^\/mnt\/([a-zA-Z])(\/.*)?$/.exec(absPath);
  return m ? m[1].toUpperCase() + ":" + (m[2] || "/") : absPath;
}

function joinPath(root, rel) {
  return root.replace(/\/+$/, "") + "/" + rel.replace(/^\.?\/+/, "");
}

// Encode only what breaks a URL path; editors choke on %5B etc. for [brackets].
function encodePath(p) {
  return p.replace(/%/g, "%25").replace(/ /g, "%20").replace(/#/g, "%23").replace(/\?/g, "%3F");
}

function editorHref(editor, ref) {
  var abs = joinPath(ref.root, ref.path);
  var pos = ":" + (ref.line || 1) + ":" + (ref.col || 1);
  var distro = encodeURIComponent(ref.distro || "Ubuntu");
  switch (editor) {
    case "vscode-wsl":
      return "vscode://vscode-remote/wsl+" + distro + encodePath(abs) + pos;
    case "vscode-win":
      return "vscode://file/" + encodePath(winPath(abs)).replace(/^\/+/, "") + pos;
    case "zed":
      return "zed://file" + encodePath(abs) + pos;
    case "cursor-wsl":
      return "cursor://vscode-remote/wsl+" + distro + encodePath(abs) + pos;
    default:
      return null;
  }
}

function copyText(ref) {
  return ref.path + ":" + (ref.line || 1);
}
