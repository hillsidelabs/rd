/**
 * Server Sent Events (SSE) extension for HTMX
 * Adapted from https://unpkg.com/htmx-ext-sse@2.2.2/dist/sse.js
 */
(function() {
  let api = {
    createClient: function(url) {
      const client = new EventSource(url);
      client.onerror = function(e) {
        console.error("SSE Error", e);
        htmx.trigger(htmx.find("[sse-connect]"), "htmx:sseError", {
          error: e,
          source: client
        });
      };
      return client;
    },
    processSSESwap: function(elt, message) {
      const detail = { elt: elt };
      if (!htmx.triggerEvent(elt, "htmx:sseBeforeMessage", detail)) return;
      
      const settleInfo = htmx.makeSettleInfo(elt);
      htmx.selectAndSwap(elt, message.data, settleInfo);
      
      if (settleInfo.tasks && settleInfo.tasks.length > 0) {
        htmx.settleImmediately(settleInfo.tasks);
      }
      
      htmx.triggerEvent(elt, "htmx:sseMessage", message);
    }
  };

  htmx.defineExtension("sse", {
    onEvent: function(name, evt) {
      if (name === "htmx:beforeCleanupElement") {
        const parent = evt.target;
        if (parent.sseClient) {
          parent.sseClient.close();
        }
      }
      return true;
    },
    
    init: function(apiRef) {
      // Store reference to internal API
      api = apiRef;
    },
    
    transformResponse: function(text, xhr, elt) {
      return text;
    },
    
    onLoad: function(elt) {
      const sseURL = elt.getAttribute("sse-connect");
      if (sseURL) {
        const client = api.createClient(sseURL);
        elt.sseClient = client;
        
        // Handle swap events
        const sseSwapAttr = elt.getAttribute("sse-swap");
        if (sseSwapAttr) {
          const swapEvents = sseSwapAttr.split(",").map(e => e.trim());
          for (const eventName of swapEvents) {
            client.addEventListener(eventName, function(message) {
              api.processSSESwap(elt, message);
            });
          }
        }
        
        // Handle trigger events
        const allTriggeredElements = htmx.findAll(elt, "[hx-trigger^='sse:']");
        for (const triggerElt of allTriggeredElements) {
          const triggerSpec = htmx.getTriggerSpecs(triggerElt);
          triggerSpec.forEach(function(spec) {
            if (spec.trigger.startsWith("sse:")) {
              const sseEvent = spec.trigger.substr(4);
              client.addEventListener(sseEvent, function() {
                htmx.trigger(triggerElt, spec.trigger);
                if (spec.once) {
                  client.removeEventListener(sseEvent, this);
                }
              });
            }
          });
        }
        
        // Handle close events
        const sseCloseAttr = elt.getAttribute("sse-close");
        if (sseCloseAttr) {
          client.addEventListener(sseCloseAttr, function() {
            client.close();
            htmx.trigger(elt, "htmx:sseClose", {
              type: "message"
            });
          });
        }
        
        htmx.trigger(elt, "htmx:sseOpen", {
          elt: elt,
          source: client
        });
      }
    }
  });
})();
