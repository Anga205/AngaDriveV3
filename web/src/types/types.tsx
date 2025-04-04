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

export type {RAMData, CPUData, SysInfo};