import { Router } from 'express';
import { body } from 'express-validator';
import PlaceController from '../controllers/place.controller';
import authMiddleware from '../middleware/auth';
import validate from '../middleware/validation';

const router = Router();

router.get('/', PlaceController.getAll);

router.get('/:id', PlaceController.getById);

router.post(
  '/',
  authMiddleware,
  [
    body('name').notEmpty().withMessage('Name is required'),
    body('address').notEmpty().withMessage('Address is required'),
    body('city').notEmpty().withMessage('City is required'),
    body('country').notEmpty().withMessage('Country is required'),
    body('latitude').isFloat({ min: -90, max: 90 }).withMessage('Valid latitude is required'),
    body('longitude').isFloat({ min: -180, max: 180 }).withMessage('Valid longitude is required'),
    validate
  ],
  PlaceController.create
);

router.put('/:id', authMiddleware, PlaceController.update);

router.delete('/:id', authMiddleware, PlaceController.delete);

router.post(
  '/:id/features',
  authMiddleware,
  [
    body('feature_id').isInt().withMessage('Valid feature_id is required'),
    validate
  ],
  PlaceController.addFeature
);

router.delete('/:id/features/:featureId', authMiddleware, PlaceController.removeFeature);

export default router;
