import { Router } from 'express';
import { body } from 'express-validator';
import AuthController from '../controllers/auth.controller';
import authMiddleware from '../middleware/auth';
import validate from '../middleware/validation';

const router = Router();

router.post(
  '/register',
  [
    body('email').isEmail().withMessage('Valid email is required'),
    body('password').isLength({ min: 6 }).withMessage('Password must be at least 6 characters'),
    body('full_name').notEmpty().withMessage('Full name is required'),
    validate
  ],
  AuthController.register
);

router.post(
  '/login',
  [
    body('email').isEmail().withMessage('Valid email is required'),
    body('password').notEmpty().withMessage('Password is required'),
    validate
  ],
  AuthController.login
);

router.get('/profile', authMiddleware, AuthController.getProfile);

export default router;
