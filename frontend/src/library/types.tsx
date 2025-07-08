interface RAMData {
    total_ram: number;
    used_ram: number;
    available_ram: number;
    ram_percent_used: number;
}

interface CPUData {
    cpu_model_name: string;
    cpu_usage: number;
}

interface SysInfo {
    ram: RAMData;
    cpu: CPUData;
}

interface GraphData {
    x_axis: Array<string>;
    y_axis: Array<number>;
    label: string;
    begin_at_zero: boolean;
}

interface IncomingData {
    type: string;
    data: SysInfo | GraphData | number;
}

type SocketStatus = "connecting" | "connected" | "disconnected" | "error" | "reconnecting";

type Pages = "Home" | "Files" | "Collections" | "Account" | "Collection";

interface FileData {
    original_file_name: string;
    file_directory: string;
    file_size: number;
    timestamp: number;
}

interface CollectionCardData {
    id: string;
    name: string;
    size: number;
    file_count: number;
    folder_count: number;
    editor_count: number;
    timestamp: number;
}

interface CollectionData {
    name: string;
    files: Array<FileData>;
    folders: Array<CollectionCardData>
    isOwned: boolean;
}

type KnownCollections = {
    [id: string]: CollectionData;
}

type AppContextType = {
    files: () => Array<FileData>;
    setFiles: (value: Array<FileData> | ((prev: Array<FileData>) => Array<FileData>)) => void;
    userCollections: () => Array<CollectionCardData>;
    setUserCollections: (value: Array<CollectionCardData> | ((prev: Array<CollectionCardData>) => Array<CollectionCardData>)) => void;
    knownCollections: () => KnownCollections;
    setKnownCollections: (value: KnownCollections | ((prev: KnownCollections) => KnownCollections)) => void;
};

export type {RAMData, CPUData, SysInfo, GraphData, IncomingData, SocketStatus, Pages, FileData, CollectionCardData, AppContextType, KnownCollections};