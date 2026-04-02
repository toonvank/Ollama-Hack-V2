// Plan Create Request
export interface PlanCreate {
  name: string;
  description?: string;
  rpm: number;
  rpd: number;
  is_default?: boolean;
}

// Plan Response
export interface PlanResponse extends PlanCreate {
  id: number;
}

// Plan Update Request
export interface PlanUpdate {
  name?: string;
  description?: string;
  rpm?: number;
  rpd?: number;
  is_default?: boolean;
}
