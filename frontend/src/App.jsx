import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Clock3, Plus, X } from "lucide-react";
import { GreetService } from "../bindings/changeme";

const RECENT_KEY = "wtw:recent-stickers";
const RECENT_MAX = 16;
const INITIAL_RENDER_COUNT = 24;
const RENDER_BATCH_SIZE = 16;
const RENDER_BATCH_DELAY_MS = 40;

const stickerIdentity = (sticker) => {
  if (!sticker || typeof sticker !== "object") return "";
  const id = sticker.id || sticker.ID;
  if (id) return String(id);
  const packID = sticker.packId || sticker.PackID || "";
  const name = sticker.name || sticker.Name || "";
  if (packID || name) return `${packID}::${name}`;
  const dataURL = sticker.dataUrl || sticker.DataURL || "";
  return dataURL ? String(dataURL) : "";
};

const normalizeSticker = (sticker) => ({
  id: sticker.id ?? sticker.ID ?? "",
  packId: sticker.packId ?? sticker.PackID ?? "",
  name: sticker.name ?? sticker.Name ?? "",
  dataUrl: sticker.dataUrl ?? sticker.DataURL ?? "",
});

const dedupeRecentStickers = (items) => {
  const unique = [];
  const seen = new Set();
  for (const raw of items) {
    const sticker = normalizeSticker(raw);
    const key = stickerIdentity(sticker);
    if (!key || seen.has(key)) continue;
    seen.add(key);
    unique.push(sticker);
    if (unique.length >= RECENT_MAX) break;
  }
  return unique;
};

function App() {
  const [packs, setPacks] = useState([]);
  const [activeNavId, setActiveNavId] = useState("");
  const [stickers, setStickers] = useState([]);
  const [recentStickers, setRecentStickers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [packLoading, setPackLoading] = useState(false);
  const [renderLimit, setRenderLimit] = useState(INITIAL_RENDER_COUNT);
  const [isPasting, setIsPasting] = useState(false);
  const navStripRef = useRef(null);
  const packLoadRequestRef = useRef(0);

  useEffect(() => {
    const raw = localStorage.getItem(RECENT_KEY);
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        setRecentStickers(dedupeRecentStickers(parsed));
      }
    } catch (err) {
      console.error(err);
    }
  }, []);

  useEffect(() => {
    const syncPopupHeight = () => {
      document.documentElement.style.setProperty("--popup-height-real", `${window.innerHeight}px`);
    };
    syncPopupHeight();
    window.addEventListener("resize", syncPopupHeight);
    return () => window.removeEventListener("resize", syncPopupHeight);
  }, []);

  useEffect(() => {
    GreetService.ListStickerPacks()
      .then((packList) => {
        setPacks(packList);
        if (packList.length > 0) {
          setActiveNavId(packList[0].id);
        }
      })
      .catch((err) => console.error(err))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (activeNavId !== "recent") return;
    setStickers(recentStickers);
  }, [activeNavId, recentStickers]);

  useEffect(() => {
    if (!activeNavId) {
      setStickers([]);
      setPackLoading(false);
      return;
    }
    if (activeNavId === "recent") {
      setPackLoading(false);
      return;
    }
    const requestID = packLoadRequestRef.current + 1;
    packLoadRequestRef.current = requestID;
    setPackLoading(true);
    setStickers([]);

    GreetService.GetPackStickers(activeNavId)
      .then((items) => {
        if (requestID !== packLoadRequestRef.current) return;
        setStickers(items);
      })
      .catch((err) => {
        if (requestID !== packLoadRequestRef.current) return;
        console.error(err);
        setStickers([]);
      })
      .finally(() => {
        if (requestID !== packLoadRequestRef.current) return;
        setPackLoading(false);
      });
  }, [activeNavId]);

  useEffect(() => {
    setRenderLimit(Math.min(stickers.length, INITIAL_RENDER_COUNT));
    if (stickers.length <= INITIAL_RENDER_COUNT) return undefined;

    let timerID = 0;
    const enqueueNextBatch = () => {
      setRenderLimit((current) => {
        const next = Math.min(stickers.length, current + RENDER_BATCH_SIZE);
        if (next < stickers.length) {
          timerID = window.setTimeout(enqueueNextBatch, RENDER_BATCH_DELAY_MS);
        }
        return next;
      });
    };

    timerID = window.setTimeout(enqueueNextBatch, RENDER_BATCH_DELAY_MS);
    return () => {
      if (timerID) {
        window.clearTimeout(timerID);
      }
    };
  }, [stickers]);

  const navItems = useMemo(() => {
    const items = [];
    if (recentStickers.length > 0) {
      items.push({
        id: "recent",
        kind: "recent",
        title: "Recent",
      });
    }
    for (const pack of packs) {
      items.push({
        id: pack.id,
        kind: "pack",
        thumbDataUrl: pack.thumbDataUrl,
        title: pack.name,
      });
    }
    return items;
  }, [packs, recentStickers]);

  const activeNavIndex = useMemo(() => navItems.findIndex((item) => item.id === activeNavId), [navItems, activeNavId]);
  const visibleStickers = useMemo(() => stickers.slice(0, renderLimit), [stickers, renderLimit]);

  const setActiveByIndex = (nextIndex) => {
    if (nextIndex < 0 || nextIndex >= navItems.length) return;
    setActiveNavId(navItems[nextIndex].id);
  };

  const persistRecent = (items) => {
    const trimmed = dedupeRecentStickers(items);
    setRecentStickers(trimmed);

    let writable = trimmed;
    while (writable.length > 0) {
      try {
        localStorage.setItem(RECENT_KEY, JSON.stringify(writable));
        return;
      } catch (err) {
        console.error("Persist recent stickers failed, trimming history", err);
        writable = writable.slice(0, -1);
      }
    }
    localStorage.removeItem(RECENT_KEY);
  };

  const onStickerClick = async (sticker) => {
    if (isPasting) return;
    setIsPasting(true);
    const selectedKey = stickerIdentity(sticker);
    const nextRecent = [sticker, ...recentStickers.filter((s) => stickerIdentity(s) !== selectedKey)];
    try {
      persistRecent(nextRecent);
      await GreetService.PasteSticker(sticker.dataUrl);
    } catch (err) {
      console.error(err);
      hidePopup();
    } finally {
      setIsPasting(false);
    }
  };

  const onRecentNavWheel = (event) => {
    if (event.deltaY > 0) {
      setActiveByIndex(activeNavIndex + 1);
    } else if (event.deltaY < 0) {
      setActiveByIndex(activeNavIndex - 1);
    }
  };

  const hidePopup = () => {
    GreetService.HidePopup().catch((err) => {
      console.error(err);
    });
  };

  useEffect(() => {
    if (!navStripRef.current || !activeNavId) return;
    const activeButton = navStripRef.current.querySelector(`[data-nav-id="${CSS.escape(activeNavId)}"]`);
    if (!activeButton) return;
    activeButton.scrollIntoView({ behavior: "smooth", inline: "nearest", block: "nearest" });
  }, [activeNavId]);

  const onNavStripWheel = (event) => {
    if (!navStripRef.current) return;
    const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY;
    if (delta === 0) return;
    event.preventDefault();
    navStripRef.current.scrollBy({ left: delta, behavior: "auto" });
  };

  return (
    <div className="popup-shell">
      <div className="popup-header">
        <div className="popup-title">Sticker Picker</div>
        <button className="popup-close" onClick={hidePopup} aria-label="Close popup">
          <X />
        </button>
      </div>

      <div className="popup-grid">
        {loading || packLoading ? (
          <div className="empty-state">Loading stickers...</div>
        ) : stickers.length === 0 ? (
          <div className="empty-state">No stickers</div>
        ) : (
          visibleStickers.map((sticker) => (
            <button
              key={sticker.id}
              className="sticker-cell"
              type="button"
              disabled={isPasting}
              onClick={() => onStickerClick(sticker)}
              title={sticker.name}
            >
              <img src={sticker.dataUrl} alt={sticker.name} loading="lazy" decoding="async" />
            </button>
          ))
        )}
      </div>

      <div className="popup-nav">
        <button
          className="nav-btn"
          type="button"
          onClick={() => setActiveByIndex(activeNavIndex - 1)}
          disabled={activeNavIndex <= 0}
          title="Back one pack"
        >
          <ChevronLeft />
        </button>

        <div className="nav-strip" ref={navStripRef} onWheel={onNavStripWheel}>
          {navItems.map((item) => (
            <button
              key={item.id}
              data-nav-id={item.id}
              type="button"
              className={`nav-pack ${item.id === activeNavId ? "active" : ""}`}
              onClick={() => setActiveNavId(item.id)}
              disabled={isPasting}
              onWheel={item.id === "recent" ? onRecentNavWheel : undefined}
              title={item.title}
            >
              {item.kind === "recent" ? (
                <span className="nav-recent-icon" aria-hidden="true">
                  <Clock3 />
                </span>
              ) : (
                <img src={item.thumbDataUrl} alt={item.title} loading="lazy" decoding="async" />
              )}
            </button>
          ))}
        </div>

        <button
          className="nav-btn"
          type="button"
          onClick={() => setActiveByIndex(activeNavIndex + 1)}
          disabled={activeNavIndex < 0 || activeNavIndex >= navItems.length - 1}
          title="Next one pack"
        >
          <ChevronRight />
        </button>

        <button className="nav-btn add-btn" type="button" title="Add sticker pack">
          <Plus />
        </button>
      </div>
    </div>
  );
}

export default App;
