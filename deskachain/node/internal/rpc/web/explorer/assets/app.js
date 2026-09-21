(function () {
  "use strict";

  var app = document.getElementById("app");
  var form = document.getElementById("search-form");
  var input = document.getElementById("search-input");
  var nav = Array.prototype.slice.call(document.querySelectorAll(".nav a"));
  var params = new URLSearchParams(window.location.search);
  var apiBase = (params.get("api") || "").replace(/\/+$/, "");
  var blockOffset = 0;
  var blockLimit = 20;
  var addressTxOffset = 0;
  var addressTxLimit = 20;
  var stakeOffset = 0;
  var stakeLimit = 20;
  var serviceOffset = 0;
  var serviceLimit = 20;
  var assetOffset = 0;
  var assetLimit = 20;

  function api(path) {
    return apiBase + path;
  }

  function escapeHTML(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return {"&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;"}[ch];
    });
  }

  function shortValue(value) {
    value = String(value == null ? "" : value);
    if (value.length <= 18) return value;
    return value.slice(0, 10) + "..." + value.slice(-8);
  }

  function badge(value, fallback) {
    var text = value == null || value === "" ? (fallback || "-") : String(value);
    return '<span class="pill tx-badge">' + escapeHTML(text) + '</span>';
  }

  function copyButton(value) {
    if (!value) return "";
    return '<button class="copy" type="button" data-copy="' + escapeHTML(value) + '" title="Copy full value">Copy</button>';
  }

  function linkHash(kind, value) {
    if (!value) return '<span class="muted">-</span>';
    return '<a class="hash" title="' + escapeHTML(value) + '" href="#/' + kind + "/" + encodeURIComponent(value) + '">' + escapeHTML(shortValue(value)) + "</a>" + copyButton(value);
  }

  function linkAddress(value) {
    if (!value) return '<span class="muted">-</span>';
    return '<a class="hash" title="' + escapeHTML(value) + '" href="#/address/' + encodeURIComponent(value) + '">' + escapeHTML(shortValue(value)) + "</a>" + copyButton(value);
  }

  function timestamp(value) {
    if (!value) return "-";
    var numeric = Number(value);
    if (!Number.isFinite(numeric)) return escapeHTML(value);
    var ms = numeric > 1000000000000 ? numeric : numeric * 1000;
    return new Date(ms).toLocaleString();
  }

  function metric(label, value) {
    return '<div class="metric"><span>' + escapeHTML(label) + '</span><strong>' + escapeHTML(value == null || value === "" ? "-" : value) + "</strong></div>";
  }

  function panel(title, body) {
    return '<section class="panel"><h1>' + escapeHTML(title) + "</h1>" + body + "</section>";
  }

  function setLoading(title) {
    app.innerHTML = panel(title || "Loading", '<p class="muted">Fetching read-only explorer data...</p>');
  }

  function setError(message) {
    app.innerHTML = panel("Explorer error",
      '<div class="error-state">' +
      '<p class="error">' + escapeHTML(message) + "</p>" +
      '<div class="toolbar"><button type="button" class="primary" data-refresh="dashboard">Back to dashboard</button></div>" +
      "</div>");
  }

  function friendlyError(err, fallback) {
    var message = err && err.message ? err.message : fallback;
    if (/block_not_found|block not found/i.test(message)) return "Block not found.";
    if (/tx_not_found|transaction not found/i.test(message)) return "Transaction not found.";
    if (/invalid_address|wrong network/i.test(message)) return "Invalid address for this network.";
    if (/not_found|no explorer result/i.test(message)) return "No explorer result found.";
    return message || fallback || "Request failed.";
  }

  function jsonFetch(path) {
    return fetch(api(path), {headers: {"Accept": "application/json"}}).then(function (res) {
      return res.text().then(function (text) {
        var data = {};
        if (text) {
          try {
            data = JSON.parse(text);
          } catch (err) {
            throw new Error("Invalid JSON response from " + path);
          }
        }
        if (!res.ok) {
          throw new Error(data.message || data.error || (res.status + " " + res.statusText));
        }
        return data;
      });
    });
  }

  function table(headers, rows, emptyText) {
    if (!rows.length) return '<p class="muted">' + escapeHTML(emptyText || "No records found.") + "</p>";
    return '<div class="table-wrap"><table><thead><tr>' + headers.map(function (h) {
      return "<th>" + escapeHTML(h) + "</th>";
    }).join("") + "</tr></thead><tbody>" + rows.join("") + "</tbody></table></div>";
  }

  function row(cells) {
    return "<tr>" + cells.map(function (cell) { return "<td>" + cell + "</td>"; }).join("") + "</tr>";
  }

  function pagerHTML(scope, data, limit) {
    var prevDisabled = data.prev_offset == null ? " disabled" : "";
    var nextDisabled = data.next_offset == null ? " disabled" : "";
    return '<div class="toolbar"><div class="pager">' +
      '<button type="button" data-page="' + scope + ':prev"' + prevDisabled + ">Previous</button>" +
      '<button type="button" data-page="' + scope + ':next"' + nextDisabled + ">Next</button>" +
      '<label>Limit <select data-limit="' + scope + '">' +
      '<option>10</option><option>20</option><option>50</option><option>100</option>' +
      '</select></label>' +
      '<span class="muted">Offset ' + escapeHTML(data.offset || 0) + " - showing " + escapeHTML(data.count || 0) + " of " + escapeHTML(data.total_count == null ? data.count || 0 : data.total_count) + "</span>" +
      "</div></div>";
  }

  function selectLimit(scope, value) {
    var select = document.querySelector('[data-limit="' + scope + '"]');
    if (select) select.value = String(value);
  }

  function parseHashQuery(value) {
    var split = String(value || "").split("?");
    return {path: split[0], query: new URLSearchParams(split[1] || "")};
  }

  function routeFromExplorerPath(path) {
    var marker = "/explorer-ui/";
    var idx = path.indexOf(marker);
    if (idx >= 0) {
      return path.slice(idx + marker.length) || "#/";
    }
    return path.charAt(0) === "#" ? path : "#/";
  }

  function updateNav() {
    var hash = window.location.hash || "#/";
    nav.forEach(function (item) {
      var active = item.getAttribute("href") === hash || (item.getAttribute("href") !== "#/" && hash.indexOf(item.getAttribute("href")) === 0);
      item.classList.toggle("active", active);
    });
  }

  function dashboardSection(title, metrics) {
    return panel(title, '<div class="grid">' + metrics + "</div>");
  }

  function renderDashboard() {
    setLoading("Dashboard");
    Promise.all([jsonFetch("/explorer/status"), jsonFetch("/health"), jsonFetch("/explorer/indexer/stats")]).then(function (results) {
      var s = results[0];
      var health = results[1];
      var indexer = results[2].stats || {};
      var ready = indexer.ready === true;
      var healthOK = health.ok === true;
      var lag = Number(indexer.lag || 0);
      var statusClass = ready && healthOK && lag === 0 ? "status-ok" : "status-warn";
      var statusText = ready && healthOK && lag === 0 ? "Synced" : (ready ? "Catching up" : "Indexer not ready");
      var summary = '<div class="dashboard-summary">' +
        '<div class="status-card ' + statusClass + '">' +
        '<span class="muted">Explorer status</span><strong>' + escapeHTML(statusText) + '</strong>' +
        '<small>Health ' + escapeHTML(String(health.ok)) + ' · Indexer height ' + escapeHTML(indexer.indexed_height) + ' · Chain height ' + escapeHTML(indexer.chain_height) + '</small>' +
        '</div>' +
        '<button class="primary" type="button" data-refresh="dashboard">Refresh</button>' +
        '</div>';

      var chainMetrics = [
        metric("Network", s.network),
        metric("Network ID", s.network_id),
        metric("Chain ID", s.chain_id),
        metric("Height", s.height),
        metric("Tip hash", s.tip_hash),
        metric("Difficulty", s.difficulty),
        metric("Next difficulty", s.next_difficulty),
        metric("Total supply", s.total_supply),
        metric("Circulating supply", s.circulating_supply)
      ].join("");

      var networkMetrics = [
        metric("Pending tx", s.pending_tx_count),
        metric("Peers", s.peer_count),
        metric("Public RPC", s.public_rpc),
        metric("Wallet RPC", s.wallet_rpc),
        metric("Admin RPC", s.admin_rpc),
        metric("Mainnet available", s.mainnet_available),
        metric("Health", health.ok)
      ].join("");

      var indexerMetrics = [
        metric("Indexer ready", indexer.ready),
        metric("Indexed height", indexer.indexed_height),
        metric("Chain height", indexer.chain_height),
        metric("Indexer lag", indexer.lag),
        metric("Indexed blocks", indexer.block_count),
        metric("Indexed transactions", indexer.transaction_count),
        metric("Address histories", indexer.address_history_count),
        metric("Asset events", indexer.asset_event_count)
      ].join("");

      var syncMetrics = [
        metric("Sync count", indexer.sync_count),
        metric("Sync failures", indexer.sync_failure_count),
        metric("Last sync duration", String(indexer.last_sync_duration_ms || 0) + " ms"),
        metric("Blocks / second", Number(indexer.blocks_per_second || 0).toFixed(2)),
        metric("Last sync", timestamp(indexer.last_sync_at_unix))
      ].join("");

      var syncError = indexer.last_sync_error ? '<p class="error">Latest indexer sync error: ' + escapeHTML(indexer.last_sync_error) + "</p>" : "";
      app.innerHTML = panel("Dashboard", summary) +
        dashboardSection("Chain", chainMetrics) +
        dashboardSection("Network & RPC", networkMetrics) +
        dashboardSection("Explorer indexer", indexerMetrics) +
        dashboardSection("Indexer sync", syncMetrics + syncError);
    }).catch(setError);
  }

  function renderBlocks() {
    setLoading("Blocks");
    jsonFetch("/explorer/blocks?limit=" + blockLimit + "&offset=" + blockOffset).then(function (data) {
      var rows = (data.blocks || []).map(function (b) {
        return row([
          '<a href="#/block/' + encodeURIComponent(b.height) + '">' + escapeHTML(b.height) + "</a>",
          linkHash("block", b.hash),
          timestamp(b.timestamp),
          escapeHTML(b.tx_count),
          linkAddress(b.miner_address),
          escapeHTML(b.reward),
          escapeHTML(b.difficulty)
        ]);
      });
      app.innerHTML = panel("Blocks", pagerHTML("blocks", data, blockLimit) + table(["Height", "Hash", "Time", "Txs", "Miner", "Reward", "Difficulty"], rows, "No blocks found."));
      selectLimit("blocks", blockLimit);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load blocks.")); });
  }

  function renderMining() {
    setLoading("Mining");
    jsonFetch("/mining/blocks?limit=10").then(function (data) {
      var metrics = [
        metric("Height", data.height),
        metric("Current difficulty", data.current_difficulty),
        metric("Next difficulty", data.next_difficulty),
        metric("Blocks until retarget", data.blocks_until_retarget),
        metric("Target block time", String(data.target_block_time_seconds || 0) + "s"),
        metric("Average interval", Number(data.average_interval_seconds || 0).toFixed(2) + "s"),
        metric("Last block age", String(data.last_block_age_seconds || 0) + "s"),
        metric("Retarget direction", data.projected_retarget_direction),
        metric("Pending tx", data.pending_tx_count),
        metric("Peers", data.peer_count)
      ].join("");
      var rows = (data.blocks || []).map(function (b) {
        return row([
          '<a href="#/block/' + encodeURIComponent(b.height) + '">' + escapeHTML(b.height) + "</a>",
          linkHash("block", b.hash),
          timestamp(b.timestamp),
          escapeHTML(b.interval_seconds),
          escapeHTML(b.difficulty),
          escapeHTML(b.tx_count),
          linkAddress(b.miner_address)
        ]);
      });
      app.innerHTML = panel("Mining / Difficulty", '<div class="grid">' + metrics + "</div><p class=\"muted\">Difficulty observation is informational only. Testnet dIDR has no monetary value.</p>") +
        panel("Recent mined blocks", table(["Height", "Hash", "Time", "Interval", "Difficulty", "Txs", "Miner"], rows, "No mined blocks found."));
    }).catch(function (err) { setError(friendlyError(err, "Unable to load mining metrics.")); });
  }

  function renderBlock(id) {
    setLoading("Block");
    var isHeight = /^\d+$/.test(id);
    jsonFetch(isHeight ? "/explorer/blocks/" + id : "/explorer/block/" + encodeURIComponent(id)).then(function (b) {
      var metrics = [
        metric("Height", b.height),
        metric("Hash", b.hash),
        metric("Previous hash", b.previous_hash),
        metric("Time", timestamp(b.timestamp)),
        metric("Difficulty", b.difficulty),
        metric("Nonce", b.nonce),
        metric("Tx count", b.tx_count),
        metric("Confirmations", b.confirmations)
      ].join("");
      var rows = (b.transactions || []).map(function (tx, index) {
        return row([
          escapeHTML(index + 1),
          linkHash("tx", tx.txid),
          badge(tx.status, "confirmed"),
          badge(tx.type),
          linkAddress(tx.from),
          linkAddress(tx.to),
          escapeHTML(tx.amount),
          escapeHTML(tx.fee)
        ]);
      });
      var height = Number(b.height);
      var navigation = '<div class="toolbar block-navigation">' +
        '<button type="button" data-block-nav="' + (Number.isFinite(height) && height > 0 ? height - 1 : "") + '"' + (height > 0 ? "" : " disabled") + '>Previous block</button>' +
        '<span class="muted">Block ' + escapeHTML(b.height) + '</span>' +
        '<button type="button" data-block-nav="' + (Number.isFinite(height) ? height + 1 : "") + '">Next block</button>' +
        '</div>';
      app.innerHTML = panel("Block detail", navigation + '<div class="grid">' + metrics + "</div>") +
        panel("Transactions", table(["#", "Txid", "Status", "Type", "From", "To", "Amount", "Fee"], rows, "No transactions in this block."));
    }).catch(function (err) { setError(friendlyError(err, "Unable to load block.")); });
  }

  function renderTx(txid) {
    setLoading("Transaction");
    jsonFetch("/explorer/tx/" + encodeURIComponent(txid)).then(function (tx) {
      var metrics = [
        metric("Txid", tx.txid),
        metric("Status", badge(tx.status)),
        metric("Type", badge(tx.type)),
        metric("Block height", tx.block_height),
        metric("Block hash", tx.block_hash),
        metric("Confirmations", tx.confirmations),
        metric("From", tx.from),
        metric("To", tx.to),
        metric("Amount", tx.amount),
        metric("Fee", tx.fee)
      ].join("");
      var addresses = (tx.involved_addresses || []).map(function (a) {
        return '<p>' + linkAddress(a) + "</p>";
      }).join("") || '<p class="muted">No address records.</p>';
      app.innerHTML = panel("Transaction detail", '<div class="grid">' + metrics + "</div>") +
        panel("Involved addresses", addresses);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load transaction.")); });
  }

  function renderAddress(address) {
    setLoading("Address");
    Promise.all([
      jsonFetch("/explorer/address/" + encodeURIComponent(address)),
      jsonFetch("/explorer/address/" + encodeURIComponent(address) + "/txs?limit=" + addressTxLimit + "&offset=" + addressTxOffset),
      jsonFetch("/explorer/address/" + encodeURIComponent(address) + "/stakes")
    ]).then(function (results) {
      var a = results[0];
      var txData = results[1];
      var txs = txData.transactions || [];
      var stakes = results[2].stakes || [];
      var addressHeader = '<div class="address-header">' +
        '<div><span class="muted">Address</span><div class="hash address-value">' + escapeHTML(a.address) + "</div></div>" +
        copyButton(a.address) +
        '</div>';
      var metrics = [
        metric("Confirmed balance", a.confirmed_balance),
        metric("Mature balance", a.mature_balance),
        metric("Immature balance", a.immature_balance),
        metric("Spendable balance", a.spendable_balance),
        metric("Active stake", a.active_stake),
        metric("Unlocking stake", a.unlocking_stake),
        metric("Tx count", a.tx_count),
        metric("First seen", a.first_seen_height),
        metric("Last seen", a.last_seen_height),
        metric("Service collateral eligible", a.service_collateral_eligible),
        metric("Service points", a.service_points)
      ].join("");
      var txRows = txs.map(function (tx) {
        return row([
          linkHash("tx", tx.txid),
          badge(tx.type),
          tx.block_height == null ? '<span class="muted">pending</span>' : '<a href="#/block/' + encodeURIComponent(tx.block_height) + '">' + escapeHTML(tx.block_height) + "</a>",
          escapeHTML(tx.amount_delta),
          escapeHTML(tx.confirmations)
        ]);
      });
      var stakeRows = stakes.map(function (s) {
        return row([
          '<span class="hash">' + escapeHTML(shortValue(s.stake_id)) + "</span>" + copyButton(s.stake_id),
          escapeHTML(s.amount),
          escapeHTML(s.status),
          escapeHTML(s.lock_height),
          escapeHTML(s.unlock_height),
          escapeHTML(s.release_height)
        ]);
      });
      app.innerHTML = panel("Address detail", addressHeader + '<div class="grid">' + metrics + "</div><p class=\"muted\">Service points are simulation-only and are not spendable dIDR.</p>") +
        panel("Recent transactions", pagerHTML("address", txData, addressTxLimit) + table(["Txid", "Type", "Block", "Delta", "Confirmations"], txRows, "No recent transactions for this address.")) +
        panel("Stake records", table(["Stake id", "Amount", "Status", "Lock height", "Unlock height", "Release height"], stakeRows, "Stake records not found."));
      selectLimit("address", addressTxLimit);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load address.")); });
  }

  function renderAsset(assetID) {
    setLoading("Asset");
    jsonFetch("/explorer/indexed/asset/" + encodeURIComponent(assetID) + "/txs?limit=" + assetLimit + "&offset=" + assetOffset).then(function (data) {
      var rows = (data.events || []).map(function (event) {
        return row([
          linkHash("tx", event.txid),
          '<a href="#/block/' + encodeURIComponent(event.block_height) + '">' + escapeHTML(event.block_height) + "</a>",
          linkAddress(event.from),
          linkAddress(event.to),
          escapeHTML(event.amount),
          escapeHTML(event.fee)
        ]);
      });
      app.innerHTML = panel("Asset detail",
        '<div class="grid">' +
        metric("Asset ID", data.asset_id) +
        metric("Event count", data.total_count) +
        metric("Indexer height", data.indexer && data.indexer.indexed_height) +
        metric("Indexer lag", data.indexer && data.indexer.lag) +
        "</div>") +
        panel("Asset events", pagerHTML("asset", data, assetLimit) + table(["Txid", "Block", "From", "To", "Amount", "Fee"], rows, "No indexed events found for this asset."));
      selectLimit("asset", assetLimit);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load asset events.")); });
  }

  function renderStakes() {
    setLoading("Stakes");
    jsonFetch("/explorer/stakes?limit=" + stakeLimit + "&offset=" + stakeOffset).then(function (data) {
      var rows = (data.stakes || []).map(function (s) {
        return row([
          '<span class="hash">' + escapeHTML(shortValue(s.stake_id)) + "</span>" + copyButton(s.stake_id),
          linkAddress(s.owner_address),
          escapeHTML(s.amount),
          '<span class="pill">' + escapeHTML(s.status) + "</span>",
          escapeHTML(s.lock_height),
          escapeHTML(s.unlock_height),
          escapeHTML(s.release_height)
        ]);
      });
      app.innerHTML = panel("Stake records", pagerHTML("stakes", data, stakeLimit) + table(["Stake id", "Owner", "Amount", "Status", "Lock height", "Unlock height", "Release height"], rows, "Stake records not found."));
      selectLimit("stakes", stakeLimit);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load stake records.")); });
  }

  function renderServices() {
    setLoading("Services");
    jsonFetch("/explorer/services?limit=" + serviceLimit + "&offset=" + serviceOffset).then(function (data) {
      var rows = (data.services || []).map(function (s) {
        return row([
          linkAddress(s.owner_address),
          escapeHTML(s.endpoint),
          escapeHTML(s.score),
          escapeHTML(s.points),
          escapeHTML(s.required_stake),
          escapeHTML(s.active_stake),
          '<span class="pill">' + escapeHTML(s.status) + "</span>",
          escapeHTML(s.simulation_only)
        ]);
      });
      app.innerHTML = panel("Service nodes", '<p class="muted">Service points are simulation-only and are not spendable dIDR. Service state is local to this RPC node.</p>' +
        pagerHTML("services", data, serviceLimit) + table(["Owner", "Endpoint", "Score", "Points", "Required stake", "Active stake", "Eligibility", "Simulation only"], rows, "Service records not found."));
      selectLimit("services", serviceLimit);
    }).catch(function (err) { setError(friendlyError(err, "Unable to load service records.")); });
  }

  function route() {
    updateNav();
    var parsed = parseHashQuery((window.location.hash || "#/").replace(/^#\/?/, ""));
    var hash = parsed.path;
    var parts = hash.split("/").filter(Boolean).map(decodeURIComponent);
    if (!parts.length) return renderDashboard();
    if (parts[0] === "blocks") {
      blockLimit = Number(parsed.query.get("limit")) || blockLimit;
      blockOffset = Number(parsed.query.get("offset")) || blockOffset;
      return renderBlocks();
    }
    if (parts[0] === "mining") return renderMining();
    if (parts[0] === "block" && parts[1]) return renderBlock(parts[1]);
    if (parts[0] === "tx" && parts[1]) return renderTx(parts[1]);
    if (parts[0] === "address" && parts[1]) return renderAddress(parts[1]);
    if (parts[0] === "asset" && parts[1]) {
      assetLimit = Number(parsed.query.get("limit")) || assetLimit;
      assetOffset = Number(parsed.query.get("offset")) || assetOffset;
      return renderAsset(parts[1]);
    }
    if (parts[0] === "stakes") return renderStakes();
    if (parts[0] === "services") return renderServices();
    setError("Route not found.");
  }

  function runSearch(q) {
    q = String(q || "").trim();
    if (!q) {
      setError("Enter a block height, block hash, transaction id, iND address, or indexed asset ID.");
      return;
    }
    input.value = q;
    setLoading("Search");
    jsonFetch("/explorer/indexed/search?q=" + encodeURIComponent(q)).then(function (data) {
      var results = data.results || [];
      if (!results.length) {
        setError("No explorer result found. Try a block height, block hash, transaction id, iND address, or indexed asset ID.");
        return;
      }
      window.location.hash = routeFromExplorerPath(results[0].path);
    }).catch(function (err) {
      setError(friendlyError(err, "Search failed."));
    });
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    runSearch(input.value);
  });

  document.addEventListener("click", function (event) {
    var copy = event.target.closest("[data-copy]");
    if (copy) {
      event.preventDefault();
      event.stopPropagation();
      var value = copy.getAttribute("data-copy");
      var done = function () {
        var old = copy.textContent;
        copy.textContent = "Copied";
        window.setTimeout(function () { copy.textContent = old; }, 900);
      };
      if (navigator.clipboard) {
        navigator.clipboard.writeText(value).then(done).catch(done);
      } else {
        done();
      }
    }
    var blockNav = event.target.closest("[data-block-nav]");
    if (blockNav && blockNav.getAttribute("data-block-nav")) {
      event.preventDefault();
      window.location.hash = "#/block/" + encodeURIComponent(blockNav.getAttribute("data-block-nav"));
      return;
    }
    var refresh = event.target.closest("[data-refresh]");
    if (refresh && refresh.getAttribute("data-refresh") === "dashboard") {
      event.preventDefault();
      window.location.hash = "#/";
      renderDashboard();
      return;
    }
    var page = event.target.closest("[data-page]");
    if (page) {
      var parts = page.getAttribute("data-page").split(":");
      var scope = parts[0];
      var direction = parts[1];
      if (scope === "blocks") {
        blockOffset = direction === "prev" ? Math.max(0, blockOffset - blockLimit) : blockOffset + blockLimit;
        window.location.hash = "#/blocks?limit=" + blockLimit + "&offset=" + blockOffset;
      }
      if (scope === "address") {
        addressTxOffset = direction === "prev" ? Math.max(0, addressTxOffset - addressTxLimit) : addressTxOffset + addressTxLimit;
        route();
      }
      if (scope === "stakes") {
        stakeOffset = direction === "prev" ? Math.max(0, stakeOffset - stakeLimit) : stakeOffset + stakeLimit;
        renderStakes();
      }
      if (scope === "services") {
        serviceOffset = direction === "prev" ? Math.max(0, serviceOffset - serviceLimit) : serviceOffset + serviceLimit;
        renderServices();
      }
      if (scope === "asset") {
        assetOffset = direction === "prev" ? Math.max(0, assetOffset - assetLimit) : assetOffset + assetLimit;
        route();
      }
    }
  });

  document.addEventListener("change", function (event) {
    var scope = event.target.getAttribute("data-limit");
    if (scope === "blocks") {
      blockLimit = Number(event.target.value) || 20;
      blockOffset = 0;
      window.location.hash = "#/blocks?limit=" + blockLimit + "&offset=0";
    }
    if (scope === "address") {
      addressTxLimit = Number(event.target.value) || 20;
      addressTxOffset = 0;
      route();
    }
    if (scope === "stakes") {
      stakeLimit = Number(event.target.value) || 20;
      stakeOffset = 0;
      renderStakes();
    }
    if (scope === "services") {
      serviceLimit = Number(event.target.value) || 20;
      serviceOffset = 0;
      renderServices();
    }
    if (scope === "asset") {
      assetLimit = Number(event.target.value) || 20;
      assetOffset = 0;
      route();
    }
  });

  window.addEventListener("hashchange", route);
  route();
})();
