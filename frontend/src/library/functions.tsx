import toast from 'solid-toast';
import type { AppContextType, CollectionCardData, FileData, SocketStatus } from './types';
import { Accessor } from 'solid-js';

const formatFileSize = (size: number) => {
    if (size < 1024) return `${size} B`;
    const kb = size / 1024;
    if (kb < 1024) return `${kb.toFixed(2)} KB`;
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(2)} MB`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} GB`;
};

const truncateFileName = (name: string) => {
    return name.length > 32 ? `${name.slice(0, 32)}...` : name;
};

const getFileType = (filePath: string) => {
    const ext = filePath.split('.').pop()?.toLowerCase();
    if (!ext) return "Unknown";
    if (["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff"].includes(ext)) return "Image";
    if (["mp4", "mkv", "avi", "mov", "wmv", "flv", "webm"].includes(ext)) return "Video";
    if (["mp3", "wav", "aac", "flac", "ogg", "wma", "m4a"].includes(ext)) return "Audio";
    if (["pdf"].includes(ext)) return "Document";
    if (["doc", "docx", "odt", "rtf", "txt", "md", "tex"].includes(ext)) return "Text Document";
    if (["xls", "xlsx", "ods", "csv", "tsv"].includes(ext)) return "Spreadsheet";
    if (["ppt", "pptx", "odp", "key"].includes(ext)) return "Presentation";
    if (["zip", "rar", "7z", "tar", "gz", "bz2", "xz"].includes(ext)) return "Archive";
    if (["exe", "msi", "bat", "sh", "apk", "dmg"].includes(ext)) return "Executable";
    if (["html", "htm", "css", "js", "ts", "jsx", "tsx", "json", "xml", "yaml", "yml", "c", "py"].includes(ext)) return "Code";
    if (["iso", "img", "bin", "cue"].includes(ext)) return "Disk Image";
    if (["epub", "mobi", "azw", "azw3"].includes(ext)) return "Ebook";
    return "Unknown";
};

const UniversalMessageHandler = (message: MessageEvent, ctx: AppContextType) => {
  const data = JSON.parse(message.data);
  if (data.type === "get_user_files_response") {
      ctx.setFiles(data.data.sort((a: FileData, b: FileData) => {
        if (b.timestamp === a.timestamp) {
          return a.file_directory.localeCompare(b.file_directory);
        }
        return b.timestamp - a.timestamp;
      }) || []);
  } else if (data.type === "file_update") {
      if (data.data.toggle === true) {
          ctx.setFiles((prev: FileData[]) => [data.data.File, ...prev]);
      } else if (data.data.toggle === false) {
          ctx.setFiles((prev: FileData[]) => prev.filter((file: FileData) => file.file_directory !== data.data.File.file_directory));
      }
  } else if (data.type === "get_user_collections_response") {
      const collections: CollectionCardData[] = data.data || [];
      ctx.setKnownCollectionCards(prev => {
          const newCards = { ...prev };
          for (const collection of collections) {
              newCards[collection.id] = collection;
          }
          return newCards;
      });
      const collectionIds = new Set(collections.map(c => c.id));
      ctx.setUserCollections(collectionIds);
  } else if (data.type === "collection_card_update"){
      ctx.setKnownCollectionCards(prev => {
          const cards = { ...prev };
          cards[data.data.id] = data.data;
          return cards;
      });
  } else if (data.type === "collection_update") {
      if (data.data.toggle === true) {
          console.log("Collection added:", data.data.collection);
          ctx.setKnownCollectionCards(prev => ({
            ...prev,
            [data.data.collection.id]: data.data.collection
          }));
          ctx.setUserCollections((prev: Set<string>) => new Set([data.data.collection.id, ...Array.from(prev)]));
      } else if (data.data.toggle === false) {
          ctx.setUserCollections((prev: Set<string>) => {
              const newSet = new Set(prev);
              newSet.delete(data.data.collection.id);
              return newSet;
          });
          ctx.setKnownCollections(prev => {
            const newCollections = { ...prev };
            delete newCollections[data.data.collection.id];
            return newCollections;
          });
          ctx.setKnownCollectionCards(prev => {
            const newCards = { ...prev };
            delete newCards[data.data.collection.id];
            return newCards;
          });
      }
  } else if (data.type === "error") {
      toast.error(`Error: ${data.data}`, {
        style: {
          "background-color": "#2a2a2a",
          "color": "#ffffff"
        }
      });
  } else if (data.type === "get_collection_response") {
    ctx.setKnownCollections(prev=>({
      ...prev,
      [data.data.collection_id]: {
        name: data.data.collection_name,
        files: data.data.files,
        folders: data.data.folders,
        isOwned: data.data.is_owner
      }
    }))
  } else if (data.type === "notification") {
    toast.custom((t)=>(
      <div 
        class="px-6 py-3 pr-12 rounded-sm shadow-md font-medium relative"
        style={{
          "background-color": "#2a2a2a",
          "color": "#ffffff"
        }}
        >
        <button 
          class="bg-gray-700/80 hover:bg-gray-600 flex justify-center top-1/2 -translate-y-1/2 items-center w-6 h-6 right-2.5 absolute rounded-full"
          onClick={() => toast.dismiss(t.id)}
        >
            &times;
        </button>
        {data.data}
      </div>
    ),{
      duration: 2000,
      position: "bottom-right",
    })
  } else if (data.type === "success_notification") {
    toast.success(data.data, {
      style: {
        "background-color": "#2a2a2a",
        "color": "#ffffff"
      }
    })
  } else if (data.type === "force_logout" && localStorage.getItem("email")===data.data) {
    handleLogout(()=>{}, ctx);
  }
}

function generateClientToken(): string {
    const chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890";
    const randomChoice = (arr: string, k: number): string => {
        let result = "";
        for (let i = 0; i < k; i++) {
            result += arr[Math.floor(Math.random() * arr.length)];
        }
        return result;
    };
    const part1 = randomChoice(chars, 10);
    const part2 = randomChoice(chars, 20);
    const part3 = String(Math.round(Date.now() / 1000));
    return `${part1}.${part2}.${part3}`;
}

const fetchFilesAndCollections = (ws: WebSocket) => {
  if (!localStorage.getItem("token") && !localStorage.getItem("email") && !localStorage.getItem("password")) {
    localStorage.setItem("token", generateClientToken());
  }
  ws.send(JSON.stringify({
    type: "get_user_files",
    data: {
      token: localStorage.getItem("token") || "",
      email: localStorage.getItem("email") || "",
      password: localStorage.getItem("password") || ""
    }
  }));
  ws.send(JSON.stringify({
    type: "get_user_collections",
    data: {
      token: localStorage.getItem("token") || "",
      email: localStorage.getItem("email") || "",
      password: localStorage.getItem("password") || ""
    }
  }));
}

const getCollection = (id: string, status: Accessor<SocketStatus>, socket: Accessor<WebSocket | undefined>, ctx: AppContextType) => {
    if (ctx.knownCollections()[id]) return;
    if (status() !== "connected") {
        setTimeout(() => getCollection(id, status, socket, ctx), 10);
    } else {
        socket()?.send(JSON.stringify({
            type: "get_collection",
            data: {
                id: id,
                auth: {
                    token: localStorage.getItem("token") || "",
                    email: localStorage.getItem("email") || "",
                    password: localStorage.getItem("password") || ""
                }
            }
        }));
    }
}

const handleLogout = (setIsLoggedIn: (value: boolean) => void, ctx: AppContextType) => {
    localStorage.removeItem("email");
    localStorage.removeItem("password");
    localStorage.removeItem("display_name");
    localStorage.setItem("token", generateClientToken());
    ctx.setFiles([]);
    ctx.setUserCollections(new Set());
    setIsLoggedIn(false);
    toast('Logged out successfully!', {
        icon: '↩️'
    })
};

function generateUUID() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Fallback for older browsers
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}
  
export {generateUUID, formatFileSize, truncateFileName, getFileType, UniversalMessageHandler, generateClientToken, fetchFilesAndCollections, getCollection, handleLogout};