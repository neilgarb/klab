$(init);

let $klab, $overlay, $error, $roundScores, $gameScores;
let ws;

function init() {
  $klab = $('#klab');
  $overlay = $('#overlay').show();
  $error = $('#error');
  $roundScores = $('#round_scores');
  $gameScores = $('#game_scores');

  $(document).on('mouseover', '.button:not(:disabled)', function() {
    playSound('button');
  });
  $(document).on('mouseover', '.player.your_turn .card', function() {
    playSound('card');
  });

  connect();
}

function connect() {
  let scheme = 'wss';
  if (document.location.protocol === 'http:') {
    scheme = 'ws';
  }
  ws = new WebSocket(scheme + '://' + document.location.host + '/ws');
  ws.onopen = function() {
    console.log('ws open');
    $overlay.hide();

    if (document.location.hash.startsWith('#') && document.location.hash.length === 5) {
      showJoinGame(document.location.hash.substr(1));
      document.location.hash = '';
    } else {
      showHome();
    }
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
  window.onbeforeunload = function() {};

  $gameScores.hide();

  $klab.html(`
<div class="klab-home">
  <img src="/client/logo.png" class="header home">
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
    showJoinGame('');
  })
}

function showNewGame() {
  $klab.html(`
<div class="klab-new-game">
  <img src="/client/logo.png" class="header">
  <form autocomplete="off">
    <label class="name">
      <span class="label">Your name</span>
      <input type="text" name="name">
    </label>
    
    <!--
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
    
    <div class="round-count" style="display: none">
      <span class="label">Number of rounds</span>
      <label>
        <input type="radio" name="round_count" value="9">
        6 rounds then 3 double rounds
      </label>
      <label>
        <input type="radio" name="round_count" value="12">
        9 rounds then 3 double rounds
      </label>
      <label>
        <input type="radio" name="round_count" value="15" checked>
        12 rounds then 3 double rounds
      </label>
      <label>
        <input type="radio" name="round_count" value="18">
        15 rounds then 3 double rounds
      </label>
    </div>
    
    <div class="max-score">
      <span class="label">Play until score</span>
      <label>
        <input type="radio" name="max_score" value="101">
        101
      </label>
      <label>
        <input type="radio" name="max_score" value="501">
        501
      </label>
      <label>
        <input type="radio" name="max_score" value="1001" checked>
        1001
      </label>
      <label>
        <input type="radio" name="max_score" value="1501">
        1501
      </label>
    </div>
    -->
  </forma>
  
  <div class="buttons">
    <button class="button create-game">Create game</button>
    <button class="button back">Main menu</button>
  </div>
</div>
`);
  $klab.find('input[name=name]').focus();

  // $klab.find('input[name=players]').click(function(e) {
  //   let numPlayers = +$('input[name=players]:checked').val();
  //   if (numPlayers === 2 || numPlayers === 4) {
  //     $klab.find('.round-count').hide();
  //     $klab.find('.max-score').show();
  //   } else if (numPlayers === 3) {
  //     $klab.find('.round-count').show();
  //     $klab.find('.max-score').hide();
  //   }
  // });

  $klab.find('.button.create-game').click(function(e) {
    e.preventDefault();

    ws.onmessage = function(j) {
      let msg = JSON.parse(j.data);
      switch (msg.type) {
        case 'game_lobby':
          showGame(msg.data);
          break;
        case 'error':
          showError(msg.data);
          break;
      }
    };

    // let playerCount = +($klab.find('input[name=players]:checked').val());
    sendMessage('create_game', {
      name: $klab.find('input[name=name]').val(),
      // player_count: playerCount,
      // max_score: +$klab.find('input[name=max_score]:checked').val(),
      // round_count: +$klab.find('input[name=round_count]:checked').val(),
    });
  });
  $klab.find('.button.back').click(function(e) {
    e.preventDefault();
    showHome();
  })
}

function showJoinGame(code) {
  $klab.html(`
<div class="klab-join-game">
  <img src="/client/logo.png" class="header">
  <form autocomplete="off">
    <label class="code">
      <span class="label">Game code</span>
      <input type="text" name="code" placeholder="XXXX">
    </label>
    <label class="name">
      <span class="label">Your name</span>
      <input type="text" name="name">
    </label>
    <div class="buttons">
      <button class="button join-game">Join game</button>
      <button class="button back">Main menu</button> 
    </div>
  </form>
</div>
`);

  let $code = $klab.find('input[name=code]');
  if (code) {
    $code.val(code);
  } else {
    $code.focus();
  }

  let $name = $klab.find('input[name=name]');
  if (code) {
    $name.focus();
  }

  $klab.find('.button.join-game').click(function(e) {
    e.preventDefault();

    ws.onmessage = function(j) {
      let msg = JSON.parse(j.data);
      switch (msg.type) {
        case 'game_lobby':
          showGame(msg.data);
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

// function showGameLobbyIndividual(data) {
//   $klab.html(`
// <div class="klab-game-lobby individual">
//   <img src="/client/logo.png" class="header">
//
//   <p class="label">Game code</p>
//   <p class="code">${data.code}</p>
//
//   <p class="label">Players</p>
//   <div class="players"></div>
//
//   <p class="label">Rules</p>
//   <p class="rules">${data.game_description}</p>
//
//   <div class="buttons">
//     <button class="button start" style="display: none" disabled>Start game</button>
//     <button class="button leave">Leave game</button>
//   </div>
// </div>
// `);
//
//   let $players = $klab.find('.players');
//   for (let i = 0; i < data.player_count; i++) {
//     let playerName = '⏳';
//     if (data.player_names.length > i) {
//       playerName = data.player_names[i];
//     }
//
//     $players.append(`<div class="player">${playerName}</div>`)
//   }
//
//   ws.onmessage = function(j) {
//     let msg = JSON.parse(j.data);
//     switch (msg.type) {
//       case 'game_lobby':
//         if (msg.data.player_count < 4) {
//           showGameLobbyIndividual(msg.data);
//         } else {
//           showGameLobbyTeams(msg.data);
//         }
//         break;
//       case 'game_started':
//         showGame(msg.data);
//         return;
//       case 'error':
//         showError(msg.data);
//         break;
//     }
//   };
//
//   let $start = $klab.find('.button.start');
//   if (data.host) {
//     $start.show();
//     if (data.can_start) {
//       $start.prop('disabled', false);
//
//       $start.click(function(e) {
//         e.preventDefault();
//         sendMessage('start_game', null);
//       });
//     }
//   }
//
//   $klab.find('.button.leave').click(function(e) {
//     e.preventDefault();
//
//     ws.onmessage = function() { };
//
//     sendMessage('leave_game', null);
//     showHome();
//   });
// }
//
// function showGameLobbyTeams(data) {
//   // TODO
// }

function showGame(data) {
  window.onbeforeunload = function() {
    return '';
  };

  $klab.html(`
<div class="klab-game">
  <div class="header">
    <img src="/client/logo2.png">
    <div class="divider"></div>
    <div class="actions">
      <button class="button scores" style="display: none">Scores</button>
    </div>
  </div>
  <div class="table">
    <div class="deck"></div>
    <div class="card_up"></div>
    <div class="trick" style="display: none"></div>
    <div class="bid_options" style="display: none;"></div>
    <div class="invite">
      <div class="text-wrapper">
        <span class="title">Game code</span>
        <a href="#${data.code}" class="code">${data.code}</a>
      </div>
      <button class="button start" style="display: none" disabled>Start</button>
    </div>
    <div class="trumps"></div>
    <div class="players"></div>
    <div class="send_message">
      <input type="text" name="message" placeholder="Message...">
      <button class="button send">Send</button>
    </div>
  </div>
</div>
`);

  let positions = calcPositions(data.player_names, data.name);
  let $players = $klab.find('.players');
  for (let i = 0; i < positions.length; i ++) {
    if (positions[i] === null) {
      continue;
    }
    let $player = renderPlayer(+i+1, positions[i], data.player_names[positions[i]]);
    if (data.player_names.length > 2 && i > 0) {
      $player.find('.swap').show();
    }
    $players.append($player);
  }

  if (data.host) {
    $klab.find('.button.start').show();
  }

  ws.onmessage = function(j) {
    let msg = JSON.parse(j.data);
    switch (msg.type) {
      case 'game_lobby':
        positions = rerenderPlayers(msg.data);
        break;
      case 'game_started':
        $klab.find('.invite').hide();
        $klab.find('.header .actions .button.scores').show();
        $klab.find('.player .swap').hide();
        break;
      case 'game_scores':
        $roundScores.hide();
        $klab.find('.trumps').hide();
        $klab.find('.trick').hide();
        $klab.find('.deck').hide();
        $klab.find('.card_up').hide();

        showGameScores(msg.data, true);
        break;
      case 'round_started':
        moveDealer(msg.data);
        break;
      case 'round_dealt':
        dealRound(positions[0], msg.data);
        break;
      case 'bid_request':
        showBidOptions(msg.data);
        break;
      case 'trumps':
        setTrumps(positions, msg.data);
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
      case 'bonus_awarded':
        showBonusAwarded(positions, msg.data);
        break;
      case 'round_scores':
        showRoundScores(msg.data);
        break;
      case 'game_over':
        showGameOver(msg.data);
        break;
      case 'error':
        showError(msg.data);
        break;
    }
  };

  $klab.find('.header .actions .button.scores').click(function(e) {
    e.preventDefault();
    showGameScores(null, false);
  });

  $klab.find('.send_message .button').click(function(e) {
    e.preventDefault();
    say();
  });

  $klab.find('.send_message input').keypress(function(e) {
    if (e.keyCode === 13) {
      say();
    }
  });

  $klab.find('.button.start').click(function(e) {
    e.preventDefault();
    sendMessage('start_game', {
    });
  });
}

function calcPositions(playerNames, name) {
  let idx;
  for (idx = 0; idx < playerNames.length; idx++) {
    if (playerNames[idx] === name) {
      break
    }
  }

  let positions = [idx, null, null, null];
  if (playerNames.length === 2) {
    positions[2] = (idx + 1) % 2;
  } else if (playerNames.length === 3) {
    positions[1] = (idx + 1) % 3;
    positions[2] = (idx + 2) % 3;
  } else if (playerNames.length === 4) {
    positions[1] = (idx + 1) % 4;
    positions[2] = (idx + 2) % 4;
    positions[3] = (idx + 3) % 4;
  }

  return positions;
}

function renderPlayer(i, pos, name) {
  let $player = $(`
  <div class="player player${i}" data-pos="${pos}">
    <span class="name">${name}</span>
    <div class="cards"></div>
    <div class="actions">
      <button class="button swap" style="display: none;">Swap</button>
    </div>
    <div class="extra">
      <div class="dealer" style="display: none;">Dealer</div>
      <div class="took_on" style="display: none;">Took on</div>
      <div class="prima" style="display: none;">Prima</div>
      <div class="pooled" style="display: none;">Pooled</div>
    </div>
    <div class="speech" style="display: none"></div>
    <div class="your_turn" style="display: none;">Your turn</div>
  </div>`);

  $player.find('.swap').click(function(e) {
    e.preventDefault();
    $klab.find('.player .swap').hide();
    sendMessage('swap', {
      new_position: pos,
    });
  });

  return $player;
}

function rerenderPlayers(data) {
  let positions = calcPositions(data.player_names, data.name);

  let $newPlayers = $(`<div class="players"></div>`);
  $newPlayers.append($klab.find('.player1').detach());

  for (let i = 1; i < positions.length; i ++) {
    if (positions[i] === null) {
      continue;
    }
    let name = data.player_names[positions[i]];
    let found = false;
    for (let j = 2; j <= 4; j ++) {
      let $player = $klab.find('.player' + j);
      if ($player.find('.name').html() === name) {
        $player.detach().removeClass('player' + j).addClass('player' + (+i+1)).appendTo($newPlayers);
        if (data.player_names.length > 2) {
          $player.find('.swap').show();
        }
        found = true;
        break;
      }
    }
    if (!found) {
      let $player = renderPlayer(i+1, positions[i], name);
      $newPlayers.append($player);
      if (data.player_names.length > 2) {
        $player.find('.swap').show();
      }
    }
  }
  $klab.find('.players').replaceWith($newPlayers);

  $klab.find('.button.start').prop('disabled', data.player_names.length < 2);

  return positions;
}

function moveDealer(data) {
  $klab.find('.took_on').hide();
  $klab.find('.prima').hide();
  $klab.find('.players .player .dealer').hide();
  $klab.find('.players .player[data-pos=' + data.dealer + '] .dealer').show();
  styleExtras();
}

let scores = null;

function showGameScores(data, sound) {
  if (data) {
    scores = data;
  } else {
    data = scores;
  }

  $gameScores.html(`
<h2>Scores</h2>
<div class="names"><div>Round #</div></div>
<div class="rounds"></div>
<div class="buttons">
  <button class="button close">Close</button>
</div>
`);
  $gameScores.show();

  if (data) {
    let $names = $gameScores.find('.names');
    for (let p of data.player_names) {
      $names.append(`<div>${p}</div>`);
    }

    let $rounds = $gameScores.find('.rounds');
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
    if (data.player_names.length !== 3) {
      let $total = $(`<div class="round"><div>Total</div></div>`);
      for (let t of data.total) {
        $total.append($(`<div>${t}</div>`));
      }
      $rounds.append($total);
    }
    $rounds.scrollTop($rounds[0].scrollHeight);
  }

  if (sound) {
    playSound('scores');
  }

  let $close = $gameScores.find('.close');
  $close.click(function(e) {
    e.preventDefault();
    $gameScores.hide();
  });
}

async function dealRound(idx, data) {
  let $deck = $klab.find('.deck').show().removeClass('other');
  if (+data.player_count === 4) {
    $deck.addClass('other');
  }
  $deck.html('');

  playSound('shuffle');
  for (let i = 0; i < data.deck_size; i ++) {
    let $card = $(`<div class="card down"></div>`);

    let offset = i*2;
    if (offset>=6) {
      continue
    }

    $card.css('left', '' + offset + 'px');
    $card.css('top', '' + offset + 'px');
    $deck.append($card);
  }

  await new Promise(resolve =>
    setTimeout(() => resolve(), 2000));

  let cardSleep = 200;
  if (+data.player_count === 4) {
    cardSleep = 120
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
          playSound('card');
          resolve();
        }, cardSleep);
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
          playSound('card');
          resolve();
        }, cardSleep);
      });
    }
  }

  let $cardUp = $klab.find('.card_up').html('').show();
  if (data.card_up.suit > 0 && data.card_up.rank > 0) {
    $cardUp.append(makeCard(data.card_up.suit, data.card_up.rank));
  } else {
    $cardUp.append(makeCard(data.suits[data.dealer], 1)); // 7
  }
  playSound('card');

  for (i = 0; i < data.player_count; i++) {
    let idx = (dealTo+i) % data.player_count;
    let $cards = $players.eq(idx).find('.cards');
    let extraCount = 3;
    if (+data.player_count === 4) {
      extraCount = 2;
    }
    for (let j = 0; j < extraCount; j++) {
      await new Promise(function(resolve) {
        setTimeout(function() {
          $cards.append(makeCard(null, null));
          playSound('card');
          resolve();
        }, cardSleep);
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
<div class="pool" style="display: none"><label><input type="checkbox" name="pool"> Pool</label></div>
`);
  } else {
    $bidOptions.html(`
<button class="button suit" data-suit="1">Clubs</button>
<button class="button suit" data-suit="2">Hearts</button>
<button class="button suit" data-suit="3">Spades</button>
<button class="button suit" data-suit="4">Diamonds</button>
<button class="button suit" data-suit="0" style="display:none;">No trumps</button>
<button class="button pass">Pass</button>
<div class="pool" style="display: none"><label><input type="checkbox" name="pool"> Pool</label></div>
`);
    $bidOptions.find('[data-suit=' + data.card_up.suit + ']').hide();
  }
  if (data.bimah) {
    $bidOptions.find('button[data-suit=0]').show();
    $bidOptions.find('button.pass').hide();
  }
  if (data.can_pool) {
    $bidOptions.find('.pool').show();
  }

  playSound('your_turn');

  $bidOptions.find('button.pass').click(function(e) {
    e.preventDefault();
    $bidOptions.html('');
    sendMessage('bid', {pass: true});
  });

  $bidOptions.find('button.play').click(function(e) {
    e.preventDefault();
    sendMessage('bid', {
      suit: data.card_up.suit,
      pool: $bidOptions.find('.pool input').prop('checked'),
    });
    $bidOptions.html('');
  });

  $bidOptions.find('button.suit').click(function(e) {
    e.preventDefault();
    sendMessage('bid', {
      suit: +$(this).attr('data-suit'),
      pool: $bidOptions.find('.pool input').prop('checked'),
    });
    $bidOptions.html('');
  });
}

async function setTrumps(positions, data) {
  $klab.find('.bid_options').hide();
  for (let j in positions) {
    if (+positions[j] === data.took_on) {
      if (data.prima) {
        $klab.find('.player' + (+j+1) + ' .prima').show();
      } else {
        $klab.find('.player' + (+j+1) + ' .took_on').show();
      }
      if (data.pooled) {
        $klab.find('.player' + (+j+1) + ' .pooled').show();
      }
    }
  }
  styleExtras();
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

  let $trumps = $klab.find('.trumps').show();
  $trumps.html(`Trumps: <span class="trumps-symbol trumps-symbol-${data.trumps}"></span>`);
}

function styleExtras() {
  $klab.find('.player .extra div').removeClass('first').removeClass('last');
  $klab.find('.player .extra').each(function() {
    $(this).find('div:visible:first').addClass('first');
    $(this).find('div:visible:last').addClass('last');
  });
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
    addSpeech(i, data.message);
  }
}

function showYourTurn(data) {
  let $player = $klab.find('.player1');
  $player.find('.your_turn').show();
  $player.addClass('your_turn');

  let $bidOptions = $klab.find('.bid_options').html('').show();
  let $trick = $klab.find('.trick');
  if (data.announce_bonus) {
    $bidOptions.html(`
<button class="button announce">Announce "${data.announce_bonus}"</button>
<button class="button skip">Keep schtum</button>  
`);

    $trick.addClass('have_announcement');

    $bidOptions.find('.button.announce').click(function(e) {
      sendMessage('announce_bonus', null);
      e.preventDefault();
      $bidOptions.hide();
      $trick.removeClass('have_announcement');
    });
    $bidOptions.find('.button.skip').click(function(e) {
      e.preventDefault();
      $bidOptions.hide();
      $trick.removeClass('have_announcement');
    });
  }

  playSound('your_turn');

  $player.find('.card.up').click(function(e) {
    e.preventDefault();
    $klab.find('.bid_options').hide();
    $klab.find('.trick').removeClass('have_announcement');
    $player.find('.card').removeClass('played');
    $(this).addClass('played');
    let card = {
      suit: +($(this).attr('data-suit')),
      rank: +($(this).attr('data-rank'))
    };
    sendMessage('play', {
      card: card,
    });
  });
}

function showTrick(positions, data) {
  $klab.find('.bid_options').hide();
  let $player = $klab.find('.player1');
  $player.find('.your_turn').hide();
  $player.find('.card.up').off('click');
  $player.find('.card.played').remove();
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
  playSound('card_played');

  let removeCardPlayer = (data.first_player + data.cards.length - 1) % data.player_count;
  for (let j in positions) {
    if (positions[j] === removeCardPlayer && j > 0) {
      $klab.find('.player' + (+j+1) + ' .card').eq(0).remove();
    }
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
        targetX = -screenWidth / 2;
        targetY = 0;
      } else if (+j === 2) {
        targetX = 0;
        targetY = -screenHeight / 2;
      } else {
        targetX = screenWidth / 2;
        targetY = 0;
      }
      let $trick = $klab.find('.trick');
      let $trickCards = $trick.find('.card');
      $trick.find('.card').css('transform', `translateX(${targetX}px) translateY(${targetY}px)`);
      playSound('trick_won');
      setTimeout(function() {
        $trickCards.remove();
      }, 500);
    }
  }
}

function showBonusAwarded(positions, data) {
  for (let j in positions) {
    if (positions[j] !== data.player) {
      continue;
    }

    let $cards = $klab.find('.player' + (+j+1) + ' .cards');
    if (+j === 0) {
      for (let c of data.cards) {
        $cards.find('.card[data-suit="' + c.suit + '"][data-rank="' + c.rank + '"]').addClass('bonus');
      }
      setTimeout(function() {
        $cards.find('.card').removeClass('bonus');
      }, 4000);
    } else {
      for (let c of data.cards) {
        let played = false;
        for (let t of data.current_trick) {
          if (t.suit === c.suit && t.rank === c.rank) {
            played = true;
            break;
          }
        }
        if (played) {
          continue;
        }
        let $available = $cards.find('.card:not(.bonus)');
        let idx = Math.floor(Math.random() * $available.length);
        $available.eq(idx).addClass('bonus').attr('data-suit', c.suit).attr('data-rank', c.rank).addClass('up');
      }
      setTimeout(function() {
        $cards.find('.card').removeClass('bonus').removeClass('up').attr('data-rank', '').attr('data-suit', '');
      }, 5000);
    }
  }
}

async function showRoundScores(data) {
  $roundScores.html('').show();

  $roundScores.append($(`<h2></h2>`).html(data.title));
  $roundScores.append($(`<div class="wrapper"></div>`));

  let i = 0;
  for (let p of data.player_names) {
    let $div = $(`
<div class="player_scores" data-player="${i}">
  <h3>
    ${p} - <span class="score">0</span>
    <span class="extra took_on" style="display: none">Took on</span>
    <span class="extra prima" style="display: none">Prima</span>
    <span class="extra pooled" style="display: none">Pooled</span>
  </h3>
  <div class="cards"></div>
  <div class="bonuses"></div>
</div>`);
    $roundScores.find('.wrapper').append($div);
    if (data.took_on === i) {
      if (data.prima) {
        $div.find('.prima').show();
      } else {
        $div.find('.took_on').show();
      }
      if (data.pooled) {
        $div.find('.pooled').show();
      }
    }
    i ++;
  }

  for (let i in data.player_names) {
    if (!data.scores || !data.scores[i]) {
      continue;
    }

    let cards = data.scores[i].won_cards;
    let bonuses = data.scores[i].bonuses;
    let $div = $roundScores.find('div[data-player="' + i + '"]');

    if (cards) {
      let sleep = Math.ceil(3000 / cards.length);
      if (sleep > 300) {
        sleep = 300;
      }
      for (let c of (cards || [])) {
        $div.find('.cards').append(makeCard(c.card.suit, c.card.rank));
        $div.find('.score').html(+$div.find('.score').html() + c.score);
        playSound('card');
        await new Promise(resolve => setTimeout(resolve, sleep));
      }
    }

    for (let b of (bonuses || [])) {
      $div.find('.bonuses').append($(`<span>${b.description} +${b.score}</span>`));
      $div.find('.score').html(+$div.find('.score').html() + b.score);
      playSound('bonus');
      await new Promise(resolve => setTimeout(resolve, 300));
    }
  }
}

function showGameOver(data) {
  $roundScores.hide();
  $gameScores.show();

  playSound('scores');

  $gameScores.html(`<h1>🏆 Game over 🏆</h1>`);

  let i = 1;
  for (let p of data.positions) {
    $gameScores.append($(`<div class="position-${i}">${p.player_name} - ${p.score}</div>`));
    i ++;
  }

  let $done = $(`<button class="button done">Main menu</button>`);
  $done.on('click', function(e) {
    e.preventDefault();
    showHome();
  });
  $gameScores.append($done);
}

function makeCard(suit, rank) {
  if (!suit || !rank) {
    return $(`<div class="card"></div>`);
  }

  return $(`
<div class="card up" data-suit="${suit}" data-rank="${rank}">
</div>`);
}

function addSpeech(position, message) {
  let $speech = $klab.find('.player' + (+position+1) + ' .speech').show();

  let $message = $(`<div></div>`).html(message);
  $speech.append($message);
  $speech.css('top', '' + (-27 - $speech.height()) + 'px');

  playSound('speech');

  setTimeout(function() {
    $message.remove();
    $speech.css('top', '' + (-27 - $speech.height()) + 'px');
    if ($speech.find('div').length === 0) {
      $speech.hide();
    }
  }, 5000);
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

let sounds = {};

function playSound(name) {
  let audioCtx = new (window.AudioContext || window.webkitAudioContext)();

  if (sounds[name]) {
    sounds[name][0].stop();
    let source = audioCtx.createBufferSource();
    source.buffer = sounds[name][1];
    source.connect(audioCtx.destination);
    source.loop = false;
    source.start();
    sounds[name] = [source, sounds[name][1]];
    return;
  }

  let xhr = new XMLHttpRequest();
  xhr.open('GET', '/client/' + name + '.mp3');
  xhr.responseType = 'arraybuffer';
  xhr.addEventListener('load', () => {
    let playsound = (audioBuffer) => {
      let source = audioCtx.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(audioCtx.destination);
      source.loop = false;
      source.start();

      sounds[name] = [source, audioBuffer];
    };
    audioCtx.decodeAudioData(xhr.response).then(playsound);
  });
  xhr.send();
}

function say() {
  let $message = $klab.find('.send_message input');
  let message = $message.val();
  $message.val('');
  if (message.trim() !== '') {
    sendMessage('speech', {message: message.trim()});
  }
}
