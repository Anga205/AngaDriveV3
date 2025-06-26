import SparkMD5 from 'spark-md5';
import toast from 'solid-toast';
import type { AppContextType, FileData } from './types';

async function getFileMD5(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();

    reader.onload = function (event) {
      const arrayBuffer = event.target?.result;
      if (arrayBuffer instanceof ArrayBuffer) {
        const wordArray = SparkMD5.ArrayBuffer.hash(arrayBuffer);
        resolve(wordArray);
      } else {
        reject(new Error('Failed to read file as ArrayBuffer'));
      }
    };

    reader.onerror = function () {
      reject(new Error('Error reading file'));
    };

    reader.readAsArrayBuffer(file);
  });
}

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
      if (data.data.error) {
          toast.error(`Error fetching files: ${data.data.error}`);
      } else {
          ctx.setFiles(data.data.sort((a: FileData, b: FileData) => b.timestamp - a.timestamp) || []);
      }
  } else if (data.type === "file_update") {
      if (data.data.toggle === true) {
          ctx.setFiles((prev: FileData[]) => [data.data.File, ...prev]);
      } else if (data.data.toggle === false) {
          ctx.setFiles((prev: FileData[]) => prev.filter((file: FileData) => file.file_directory !== data.data.File.file_directory));
      }
  }
}

export {getFileMD5, formatFileSize, truncateFileName, getFileType, UniversalMessageHandler};