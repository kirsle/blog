/*
Reusable script to disable the ACE Code Editor at the touch of a button, i.e.
for mobile where the editor doesn't work very well.

Include this script at the bottom of the .gohtml page and have a button with
the ID "ace-toggle-button".

It sets the global window variable DISABLE_ACE_EDITOR=true if disabled.
*/
(function() {
	let key = "ace-toggle.disabled";
	let disabled = localStorage[key] ? true : false;
	window.DISABLE_ACE_EDITOR = disabled;

	let $button = document.querySelector("#ace-toggle-button");
	if (disabled) {
		$button.innerText = "Enable Rich Code Editor";
	} else {
		$button.innerText = "Disable Rich Code Editor";
	}

	$button.addEventListener("click", function() {
		if (!window.confirm("Toggling the code editor will reload the page. Are you sure?")) {
			return false;
		}

		if (disabled) {
			delete localStorage[key];
		} else {
			localStorage[key] = true;
		}
		window.location.reload();
	});
})();
