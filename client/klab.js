$(init);

let $klab, $overlay, $error, $scores;
let ws;

function init() {
  $klab = $('#klab');
  $overlay = $('#overlay').show();
  $error = $('#error');
  $scores = $('#scores');
  connect();
}

function connect() {
  ws = new WebSocket('ws://localhost:8080/ws');
  ws.onopen = function() {
    console.log('ws open');
    $overlay.hide();
    showHome();
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
  <div class="table">
    <div class="players">
    </div>
    <div class="deck"></div>
    <div class="card_up"></div>
    <div class="bid_options" style="display: none;"></div>
    <div class="trick" style="display: none"></div>
    <div class="trumps"></div>
  </div>
</div>
`);

  let idx;
  for (idx = 0; idx < data.player_names.length; idx++) {
    if (data.player_names[idx] === data.name) {
      break
    }
  }

  let positions = [null, null, null, null];
  if (data.player_names.length === 2) {
    positions[0] = idx;
    positions[2] = (idx+1) % 2;
  } else if (data.player_names.length === 3) {
    positions[0] = idx;
    positions[1] = (idx+1) % 3;
    positions[2] = (idx+2) % 3;
  } else if (data.player_names.length === 4) {
    positions[0] = idx;
    positions[1] = (idx+1) % 4;
    positions[2] = (idx+2) % 4;
    positions[3] = (idx+3) % 4;
  }

  let $players = $klab.find('.players');
  for (let i = 0; i < positions.length; i ++) {
    if (positions[i] === null) {
      continue;
    }
    $player = $(`
<div class="player player${i+1}" data-pos="${positions[i]}">
  <span class="name">${data.player_names[positions[i]]}</span>
  <div class="cards"></div>
  <div class="dealer" style="display: none;">
    <span>Dealer</span> 
  </div>
  <div class="speech" style="display: none"></div>
  <div class="your_turn" style="display: none;">Your turn</div>
</div>
`);
    $players.append($player);
  }

  ws.onmessage = function(j) {
    let msg = JSON.parse(j.data);
    switch (msg.type) {
      case 'game_scores':
        showGameScores(msg.data);
        break;
      case 'round_started':
        $scores.hide();
        moveDealer(msg.data);
        break;
      case 'round_dealt':
        dealRound(idx, msg.data);
        break;
      case 'bid_request':
        showBidOptions(msg.data);
        break;
      case 'trumps':
        setTrumps(msg.data);
        break;
      case 'speech':
        showSpeech(positions, msg.data);
        break;
      case 'your_turn':
        showYourTurn(msg.data);
        break;
      case 'trick':
        showTrick(positions, msg.data);
        break;
      case 'trick_won':
        showTrickWon(positions, msg.data);
        break;
      case 'error':
        showError(msg.data);
        break;
    }
  };
}

function moveDealer(data) {
  $klab.find('.players .player .dealer').hide();
  $klab.find('.players .player[data-pos=' + data.dealer + '] .dealer').show();
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

async function dealRound(myIdx, data) {
  let $deck = $klab.find('.deck');
  $deck.html('');

  for (let i = 0; i < data.deck_size; i ++) {
    let $card = $(`<div class="card down"></div>`);

    let offset = i*2;
    if (offset>6) {
      continue
    }

    $card.css('left', '' + offset + 'px');
    $card.css('top', '' + offset + 'px');
    $deck.append($card);
  }

  let $players = $klab.find('.players .player');
  $players.find('.cards').html('');
  let dealTo = (data.dealer+1) % data.player_count;
  for (let i = 0; i < data.player_count; i++) {
    let idx = (dealTo+i) % data.player_count;
    let $cards = $players.eq(idx).find('.cards');
    for (let j = 0; j < 3; j++) {
      await new Promise(function(resolve) {
        setTimeout(function() {
          let $card = makeCard(null, null);
          if (idx === 0) {
            $card = makeCard(data.cards[j].suit, data.cards[j].rank);
          }
          $cards.append($card);
          resolve();
        }, 200);
      });
    }
  }
  for (i = 0; i < data.player_count; i++) {
    let idx = (dealTo+i) % data.player_count;
    let $cards = $players.eq(idx).find('.cards');
    for (let j = 0; j < 3; j++) {
      await new Promise(function(resolve) {
        setTimeout(function() {
          let $card = makeCard(null, null);
          if (idx === 0) {
            $card = makeCard(data.cards[j+3].suit, data.cards[j+3].rank);
          }
          $cards.append($card);
          resolve();
        }, 200);
      });
    }
  }

  let $cardUp = $klab.find('.card_up');
  $cardUp.append(makeCard(data.card_up.suit, data.card_up.rank));

  for (i = 0; i < data.player_count; i++) {
    let idx = (dealTo+i) % data.player_count;
    let $cards = $players.eq(idx).find('.cards');
    for (let j = 0; j < 2; j++) {
      await new Promise(function(resolve) {
        setTimeout(function() {
          $cards.append(makeCard(null, null));
          resolve();
        }, 200);
      });
    }
  }

  sortCards();
}

function showBidOptions(data) {
  let $bidOptions = $klab.find('.bid_options').show();
  if (!data.round2) {
    $bidOptions.html(`
<button class="button play">Play</button>
<button class="button pass">Pass</button>
`);
  } else {
    $bidOptions.html(`
<button class="button suit" data-suit="1">Clubs</button>
<button class="button suit" data-suit="2">Diamonds</button>
<button class="button suit" data-suit="3">Hearts</button>
<button class="button suit" data-suit="4">Spades</button>
<button class="button suit" data-suit="0" style="display:none;">No trumps</button>
<button class="button pass">Pass</button>
`);
    $bidOptions.find('[data-suit=' + data.card_up.suit + ']').hide();
  }
  if (data.bimah) {
    $bidOptions.find('button[data-suit=0]').show();
    $bidOptions.find('button.pass').hide();
  }

  $bidOptions.find('button.pass').click(function(e) {
    e.preventDefault();
    $bidOptions.html('');
    sendMessage('bid', {'pass': true});
  });

  $bidOptions.find('button.play').click(function(e) {
    e.preventDefault();
    $bidOptions.html('');
    sendMessage('bid', {'suit': data.card_up.suit});
  });

  $bidOptions.find('button.suit').click(function(e) {
    e.preventDefault();
    $bidOptions.html('');
    sendMessage('bid', {'suit': +$(this).attr('data-suit')});
  });
}

async function setTrumps(data) {
  $klab.find('.bid_options').hide();
  let $cards = $klab.find('.player1 .cards');
  $cards.find('.card:gt(5)').remove();
  for (let c of data.extra_cards) {
    await new Promise(function(resolve) {
      setTimeout(function() {
        $cards.append(makeCard(c.suit, c.rank));
        resolve();
      }, 200);
    });
  }

  sortCards();

  let $trumps = $klab.find('.trumps');
  $trumps.show().html(`Trumps: <span class="trumps-symbol trumps-symbol-${data.trumps}"></span>`);
}

function sortCards() {
  let $myCards = $klab.find('.player1 .cards');
  $myCards.find('.card').detach().sort((a, b) => {
    if ($(a).hasClass('up') === $(b).hasClass('up')) {
      if (+$(a).attr('data-suit') === +$(b).attr('data-suit')) {
        return +$(a).attr('data-rank') < +$(b).attr('data-rank') ? -1 : 1;
      }
      return +$(a).attr('data-suit') < +$(b).attr('data-suit') ? -1 : 1;
    }
    return $(a).hasClass('up') ? -1 : 1;
  }).appendTo($myCards);
}

function showSpeech(positions, data) {
  for (let i in positions) {
    if (positions[i] !== data.player) {
      continue;
    }

    let idx = +i + 1;
    let find = '.player' + idx + ' .speech';
    let $speech = $klab.find(find).html(data.message).show();
    setTimeout(function() {
      $speech.hide();
    }, 3000);
  }
}

function showYourTurn(data) {
  let $player = $klab.find('.player1');
  $player.find('.your_turn').show();
  $player.addClass('your_turn');

  $player.find('.card.up').click(function(e) {
    e.preventDefault();
    let card = {
      suit: +($(this).attr('data-suit')),
      rank: +($(this).attr('data-rank'))
    };
    sendMessage('play', {
      card: card,
    });
    $(this).remove();
  });
}

function showTrick(positions, data) {
  let $player = $klab.find('.player1');
  $player.find('.your_turn').hide();
  $player.find('.card.up').off('click');
  $player.removeClass('your_turn');
  let $trick = $klab.find('.trick').show();
  $trick.html('');
  for (let i in data.cards) {
    let $card = makeCard(data.cards[i].suit, data.cards[i].rank);
    let pos = (+i + data.first_player) % data.player_count;
    for (let j in positions) {
      if (positions[j] === pos) {
        $card.addClass('trick-player' + (+j+1));
        break;
      }
    }
    $trick.append($card);
  }
}

function showTrickWon(positions, data) {
  for (let j in positions) {
    if (positions[j] === (data.first_player + data.winner) % data.player_count)  {
      let screenWidth = $(window).width();
      let screenHeight = $(window).height();

      let targetX, targetY;
      if (+j === 0) {
        targetX = 0;
        targetY = screenHeight / 2;
      } else if (+j === 1) {
        targetX = screenWidth / 2;
        targetY = 0;
      } else if (+j === 2) {
        targetX = 0;
        targetY = -screenHeight / 2;
      } else {
        targetX = -screenWidth / 2;
        targetY = 0;
      }

      let $trick = $klab.find('.trick');
      $trick.find('.card').css('transform', `translateX(${targetX}px) translateY(${targetY}px)`);
      setTimeout(function() {
        $klab.find('.trick').html('');
        $klab.find('.card.won').remove();
      }, 500);
    }
  }
}

function makeCard(suit, rank) {
  if (!suit || !rank) {
    return $(`<div class="card"></div>`);
  }

  return $(`
<div class="card up" data-suit="${suit}" data-rank="${rank}">
</div>`);
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