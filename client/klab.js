$(init);

let $klab, $overlay, $error, $scores;
let ws;

function init() {
  $klab = $('#klab');
  $overlay = $('#overlay').show();
  $error = $('#error');
  $scores = $('#scores');
  connect();
  showHome();
}

function connect() {
  ws = new WebSocket('ws://127.0.0.1:8081/ws');
  ws.onopen = function() {
    console.log('ws open');
    $overlay.hide();
  };
  ws.onclose = function() {
    console.log('ws close');
    $overlay.show();
    setTimeout(connect, 1000);
  };
  ws.onerror = function(e) {
    console.log('ws error', e);
  };
}

function showHome() {
  $klab.html(`
<div class="klab-home">
  <img src="jack.png" class="header">
  <h1>Jassus, boet!</h1>
  <p>A multiplayer Klabberjas game</p>
  <div class="buttons">
    <button class="button new-game">New game</button>
    <button class="button join-game">Join game</button>
  </div>
</div>
`);
  $klab.find('.button.new-game').click(function(e) {
    e.preventDefault();
    showNewGame();
  })
  $klab.find('.button.join-game').click(function(e) {
    e.preventDefault();
    showJoinGame();
  })
}

function showNewGame() {
  $klab.html(`
<div class="klab-new-game">
  <img src="jack.png" class="header">
  <h1>Jassus, boet!</h1>
  <form autocomplete="off">
    <label class="name">
      <span class="label">Your name</span>
      <input type="text" name="name" required>
    </label>
    
    <div class="player-count">
      <span class="label">Number of players</span>
      <label>
        <input type="radio" name="players" value="2" checked>
        2 players
      </label>
      <label>
        <input type="radio" name="players" value="3">
        3 players
      </label>
      <label>
        <input type="radio" name="players" value="4">
        4 players
      </label>
    </div>
  </forma>
  
  <div class="buttons">
    <button class="button create-game">Create game</button>
    <button class="button back">Main menu</button>
  </div>
</div>
`);
  $klab.find('input[name=name]').focus();
  $klab.find('.button.create-game').click(function(e) {
    e.preventDefault();

    ws.onmessage = function(j) {
      let msg = JSON.parse(j.data);
      switch (msg.type) {
        case 'game_lobby':
          if (msg.data.player_count < 4) {
            showGameLobbyIndividual(msg.data);
          } else {
            showGameLobbyTeams(msg.data);
          }
          break;
        case 'error':
          showError(msg.data);
          break;
      }
    };

    let playerCount = +($klab.find('input[name=players]:checked').val());
    sendMessage('create_game', {
      name: $klab.find('input[name=name]').val(),
      player_count: playerCount,
    });
  });
  $klab.find('.button.back').click(function(e) {
    e.preventDefault();
    showHome();
  })
}

function showJoinGame() {
  $klab.html(`
<div class="klab-join-game">
  <img src="jack.png" class="header">
  <h1>Jassus, boet!</h1>
  <form autocomplete="off">
    <label class="code">
      <span class="label">Game code</span>
      <input type="text" name="code" placeholder="XXXX" pattern="[A-Z]{4}" required>
    </label>
    <label class="name">
      <span class="label">Your name</span>
      <input type="text" name="name" required>
    </label>
    <div class="buttons">
      <button class="button join-game">Join game</button>
      <button class="button back">Main menu</button> 
    </div>
  </form>
</div>
`);
  $klab.find('input[name=code]').focus();
  $klab.find('.button.join-game').click(function(e) {
    e.preventDefault();

    ws.onmessage = function(j) {
      let msg = JSON.parse(j.data);
      switch (msg.type) {
        case 'game_lobby':
          if (msg.data.player_count < 4) {
            showGameLobbyIndividual(msg.data);
          } else {
            showGameLobbyTeams(msg.data);
          }
          break;
        case 'error':
          showError(msg.data);
          break;
      }
    };

    sendMessage('join_game', {
      code: $klab.find('input[name=code]').val(),
      name: $klab.find('input[name=name]').val(),
    });
  });
  $klab.find('.button.back').click(function(e) {
    e.preventDefault();
    showHome();
  });
}

function showGameLobbyIndividual(data) {
  $klab.html(`
<div class="klab-game-lobby individual">
  <img src="jack.png" class="header">
  <h1>Jassus, boet!</h1>
  
  <p>Game code</p>
  <p class="code">${data.code}</p>
  
  <p>Players</p>
  <ol class="players"></ol>
  
  <div class="buttons">
    <button class="button start" style="display: none" disabled>Start game</button>
    <button class="button leave">Leave game</button> 
  </div>
</div>
`);

  let $players = $klab.find('.players');
  for (let i = 0; i < data.player_count; i++) {
    let playerName = '⏳';
    if (data.player_names.length > i) {
      playerName = data.player_names[i];
    }

    $players.append(`<li class="player">${playerName}</li>`)
  }

  ws.onmessage = function(j) {
    let msg = JSON.parse(j.data);
    switch (msg.type) {
      case 'game_lobby':
        if (msg.data.player_count < 4) {
          showGameLobbyIndividual(msg.data);
        } else {
          showGameLobbyTeams(msg.data);
        }
        break;
      case 'game_started':
        showGame(msg.data);
        return;
      case 'error':
        showError(msg.data);
        break;
    }
  };

  let $start = $klab.find('.button.start');
  if (data.host) {
    $start.show();
    if (data.can_start) {
      $start.prop('disabled', false);

      $start.click(function(e) {
        e.preventDefault();
        sendMessage('start_game', null);
      });
    }
  }

  $klab.find('.button.leave').click(function(e) {
    e.preventDefault();

    ws.onmessage = function() { };

    sendMessage('leave_game', null);
    showHome();
  });
}

function showGameLobbyTeams(data) {
  $klab.html(`
<div class="klab-game-lobby teams">
  <img src="jack.png" class="header">
  <h1>Jassus, boet!</h1>
  
  <p>Game code</p>
  <p class="code">${data.code}</p>
</div>
`);
}

function showGame(data) {
  $klab.html(`
<div class="klab-game">
  <div class="header">
    <img src="jack.png">
    <h1>Jassus, boet!</h1>
  </div>
  <div class="players"></div>
</div>
`);

  ws.onmessage = function(j) {
    let msg = JSON.parse(j.data);
    switch (msg.type) {
      case 'game_scores':
        showGameScores(msg.data);
        break;
      case 'error':
        showError(msg.data);
        break;
    }
  };

  let idx;
  for (idx = 0; idx < data.player_names.length; idx++) {
    if (data.player_names[idx] === data.name) {
      break
    }
  }

  let positions = [null, null, null, null];
  if (data.player_names.length === 2) {
    positions[0] = data.player_names[idx];
    positions[2] = data.player_names[(idx+1) % 2];
  } else if (data.player_names.length === 3) {
    positions[0] = data.player_names[idx];
    positions[1] = data.player_names[(idx+1) % 3];
    positions[2] = data.player_names[(idx+2) % 3];
  } else if (data.player_names.length === 4) {
    positions[0] = data.player_names[idx];
    positions[1] = data.player_names[(idx+1) % 3];
    positions[2] = data.player_names[(idx+2) % 3];
    positions[3] = data.player_names[(idx+3) % 3];
  }

  let $players = $klab.find('.players');
  for (let i = 0; i < positions.length; i ++) {
    if (positions[i] === null) {
      continue;
    }
    $player = $(`
<div class="player player${i+1}">
  <span class="name">${positions[i]}</span>
</div>
`);
    $players.append($player);
  }
}

function showGameScores(data) {
  $scores.html(`
<h2>🏆 Scores 🏆</h2>
<div class="names"><div>Round #</div></div>
<div class="rounds"></div>
<div class="buttons">
  <button class="button close">Close</button>
</div>
`);
  $scores.show();

  let $names = $scores.find('.names');
  for (let p of data.player_names) {
    $names.append(`<div>${p}</div>`);
  }

  let $rounds = $scores.find('.rounds');
  let i = 1;
  for (let r of (data.scores || [])) {
    let $round = $(`<div class="round"><div>${i}</div></div>`);
    for (let s of r) {
      let score = +s;
      $round.append(`<div>${score}</div>`);
    }
    $rounds.append($round);
    i++;
  }
  $rounds.scrollTop($rounds[0].scrollHeight);

  let $close = $scores.find('.close');
  $close.click(function(e) {
    e.preventDefault();
    $scores.hide();
  });
}

function sendMessage(typ, data) {
  ws.send(JSON.stringify({
    type: typ,
    data: data,
  }));
}

function showError(msg) {
  $error.html(`
<span class="message">${msg}</span>
<a href="">Close</a>
 `);
  $error.show();
  $error.find('a').click(function(e) {
    e.preventDefault();
    $error.hide();
  });
}