import { Request } from 'express';
import { UserResponse } from './user.types';

export interface AuthenticatedRequest extends Request {
  user: UserResponse;
}
