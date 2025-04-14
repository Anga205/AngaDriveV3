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
    beginAtZero: boolean;
}

interface IncomingData {
    type: string;
    data: SysInfo | GraphData;
}

export type {RAMData, CPUData, SysInfo, GraphData, IncomingData};