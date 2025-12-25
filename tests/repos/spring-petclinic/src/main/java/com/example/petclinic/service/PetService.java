package com.example.petclinic.service;

import com.example.petclinic.dto.PetDto;
import com.example.petclinic.exception.ResourceNotFoundException;
import com.example.petclinic.model.Owner;
import com.example.petclinic.model.Pet;
import com.example.petclinic.model.PetType;
import com.example.petclinic.repository.OwnerRepository;
import com.example.petclinic.repository.PetRepository;
import com.example.petclinic.repository.PetTypeRepository;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.stream.Collectors;

/**
 * Service layer for Pet operations.
 * Demonstrates async operations and complex business logic.
 */
@Service
@Transactional
public class PetService {

    private final PetRepository petRepository;
    private final OwnerRepository ownerRepository;
    private final PetTypeRepository petTypeRepository;

    public PetService(PetRepository petRepository,
                      OwnerRepository ownerRepository,
                      PetTypeRepository petTypeRepository) {
        this.petRepository = petRepository;
        this.ownerRepository = ownerRepository;
        this.petTypeRepository = petTypeRepository;
    }

    /**
     * Find all pets.
     */
    @Transactional(readOnly = true)
    public List<PetDto> findAll() {
        return petRepository.findAll().stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find pet by ID.
     */
    @Transactional(readOnly = true)
    public PetDto findById(Long id) {
        Pet pet = petRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Pet", "id", id));
        return toDto(pet);
    }

    /**
     * Find pets by owner ID.
     */
    @Transactional(readOnly = true)
    public List<PetDto> findByOwnerId(Long ownerId) {
        return petRepository.findByOwnerId(ownerId).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Create a new pet for an owner.
     */
    public PetDto create(PetDto petDto) {
        Owner owner = ownerRepository.findById(petDto.ownerId())
            .orElseThrow(() -> new ResourceNotFoundException("Owner", "id", petDto.ownerId()));

        Pet pet = new Pet();
        pet.setName(petDto.name());
        pet.setBirthDate(petDto.birthDate());
        pet.setOwner(owner);

        if (petDto.typeName() != null) {
            PetType type = petTypeRepository.findByName(petDto.typeName())
                .orElseThrow(() -> new ResourceNotFoundException("PetType", "name", petDto.typeName()));
            pet.setType(type);
        }

        Pet savedPet = petRepository.save(pet);
        return toDto(savedPet);
    }

    /**
     * Update an existing pet.
     */
    public PetDto update(Long id, PetDto petDto) {
        Pet pet = petRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Pet", "id", id));

        pet.setName(petDto.name());
        pet.setBirthDate(petDto.birthDate());

        if (petDto.typeName() != null) {
            PetType type = petTypeRepository.findByName(petDto.typeName())
                .orElseThrow(() -> new ResourceNotFoundException("PetType", "name", petDto.typeName()));
            pet.setType(type);
        }

        Pet savedPet = petRepository.save(pet);
        return toDto(savedPet);
    }

    /**
     * Delete a pet.
     */
    public void delete(Long id) {
        if (!petRepository.existsById(id)) {
            throw new ResourceNotFoundException("Pet", "id", id);
        }
        petRepository.deleteById(id);
    }

    /**
     * Async method to search pets by name.
     */
    @Async
    @Transactional(readOnly = true)
    public CompletableFuture<List<PetDto>> searchByNameAsync(String name) {
        List<PetDto> pets = petRepository.findByNameContainingIgnoreCase(name).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
        return CompletableFuture.completedFuture(pets);
    }

    /**
     * Get pet type statistics.
     */
    @Transactional(readOnly = true)
    public long countByType(String typeName) {
        return petRepository.countByTypeName(typeName);
    }

    private PetDto toDto(Pet pet) {
        return new PetDto(
            pet.getId(),
            pet.getName(),
            pet.getBirthDate(),
            pet.getType() != null ? pet.getType().getName() : null,
            pet.getOwner() != null ? pet.getOwner().getId() : null
        );
    }
}
