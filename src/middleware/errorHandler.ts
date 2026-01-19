import { Request, Response, NextFunction } from 'express';

interface DatabaseError extends Error {
  code?: string;
}

interface CustomError extends Error {
  statusCode?: number;
}

const errorHandler = (err: CustomError | DatabaseError, req: Request, res: Response, next: NextFunction): void => {
  console.error('Error:', err);

  if (err.name === 'ValidationError') {
    res.status(400).json({
      error: 'Validation Error',
      details: err.message
    });
    return;
  }

  if ('code' in err && err.code === '23505') {
    res.status(409).json({
      error: 'Duplicate Entry',
      message: 'Resource already exists'
    });
    return;
  }

  if ('code' in err && err.code === '23503') {
    res.status(400).json({
      error: 'Invalid Reference',
      message: 'Referenced resource does not exist'
    });
    return;
  }

  const statusCode = 'statusCode' in err ? err.statusCode || 500 : 500;
  const message = err.message || 'Internal Server Error';

  res.status(statusCode).json({
    error: message,
    ...(process.env.NODE_ENV === 'development' && { stack: err.stack })
  });
};

export default errorHandler;
