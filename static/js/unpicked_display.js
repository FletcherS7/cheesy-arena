// Copyright 2026 Team 254. All Rights Reserved.

// Client-side methods for the unpicked teams display.

$(function () {
  // Read the configuration for this display from the URL query string.
  const urlParams = new URLSearchParams(window.location.search);

  if (urlParams.get("inverted") === "true") {
    $("#unpickedTeamsContainer").css("transform", "rotate(180deg)");
  }

  const $unpickedTeams = $("#unpickedTeams");
  const $container = $("#unpickedTeamsContainer");
  const unpickedTeamsTemplate = Handlebars.compile($("#unpickedTeamsTemplate").html());

  new CheesyWebsocket("/displays/unpicked/websocket", {
    allianceSelection: function (event) {
      const data = event.data;
      const rankedTeams = data.RankedTeams;
      if (rankedTeams && rankedTeams.length > 0) {
        $container.show();
        let unpickedTeams = [];

        for (let i = 0; i < rankedTeams.length; i++) {
          if (!rankedTeams[i].Picked) {
            unpickedTeams.push(rankedTeams[i]);
          }
        }

        $unpickedTeams.html(unpickedTeamsTemplate(unpickedTeams));
      } else {
        $container.hide();
      }
    }
  });
});
