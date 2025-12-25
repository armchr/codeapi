package com.example.petclinic.service;

import com.example.petclinic.dto.OwnerDto;
import com.example.petclinic.dto.PetDto;
import com.example.petclinic.exception.ResourceNotFoundException;
import com.example.petclinic.exception.DuplicateResourceException;
import com.example.petclinic.model.Owner;
import com.example.petclinic.repository.OwnerRepository;
import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Service layer for Owner operations.
 * Demonstrates Spring caching, transactions, and business logic.
 */
@Service
@Transactional
public class OwnerService {

    private final OwnerRepository ownerRepository;

    public OwnerService(OwnerRepository ownerRepository) {
        this.ownerRepository = ownerRepository;
    }

    /**
     * Find all owners.
     */
    @Transactional(readOnly = true)
    @Cacheable("owners")
    public List<OwnerDto> findAll() {
        return ownerRepository.findAll().stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Find owner by ID.
     */
    @Transactional(readOnly = true)
    @Cacheable(value = "owners", key = "#id")
    public OwnerDto findById(Long id) {
        Owner owner = ownerRepository.findByIdWithPets(id)
            .orElseThrow(() -> new ResourceNotFoundException("Owner", "id", id));
        return toDtoWithPets(owner);
    }

    /**
     * Search owners by last name.
     */
    @Transactional(readOnly = true)
    public List<OwnerDto> findByLastName(String lastName) {
        return ownerRepository.findByLastNameContaining(lastName).stream()
            .map(this::toDto)
            .collect(Collectors.toList());
    }

    /**
     * Create a new owner.
     */
    @CacheEvict(value = "owners", allEntries = true)
    public OwnerDto create(OwnerDto ownerDto) {
        if (ownerDto.email() != null && ownerRepository.existsByEmail(ownerDto.email())) {
            throw new DuplicateResourceException("Owner", "email", ownerDto.email());
        }

        Owner owner = toEntity(ownerDto);
        Owner savedOwner = ownerRepository.save(owner);
        return toDto(savedOwner);
    }

    /**
     * Update an existing owner.
     */
    @CacheEvict(value = "owners", key = "#id")
    public OwnerDto update(Long id, OwnerDto ownerDto) {
        Owner owner = ownerRepository.findById(id)
            .orElseThrow(() -> new ResourceNotFoundException("Owner", "id", id));

        // Check for duplicate email if changed
        if (ownerDto.email() != null && !ownerDto.email().equals(owner.getEmail())
            && ownerRepository.existsByEmail(ownerDto.email())) {
            throw new DuplicateResourceException("Owner", "email", ownerDto.email());
        }

        updateEntity(owner, ownerDto);
        Owner savedOwner = ownerRepository.save(owner);
        return toDto(savedOwner);
    }

    /**
     * Delete an owner.
     */
    @CacheEvict(value = "owners", allEntries = true)
    public void delete(Long id) {
        if (!ownerRepository.existsById(id)) {
            throw new ResourceNotFoundException("Owner", "id", id);
        }
        ownerRepository.deleteById(id);
    }

    /**
     * Get statistics about owners by city.
     */
    @Transactional(readOnly = true)
    public long countByCity(String city) {
        return ownerRepository.countByCity(city);
    }

    // Mapping methods

    private OwnerDto toDto(Owner owner) {
        return new OwnerDto(
            owner.getId(),
            owner.getFirstName(),
            owner.getLastName(),
            owner.getAddress(),
            owner.getCity(),
            owner.getTelephone(),
            owner.getEmail(),
            null
        );
    }

    private OwnerDto toDtoWithPets(Owner owner) {
        List<PetDto> pets = owner.getPets().stream()
            .map(pet -> new PetDto(
                pet.getId(),
                pet.getName(),
                pet.getBirthDate(),
                pet.getType() != null ? pet.getType().getName() : null,
                owner.getId()
            ))
            .collect(Collectors.toList());

        return new OwnerDto(
            owner.getId(),
            owner.getFirstName(),
            owner.getLastName(),
            owner.getAddress(),
            owner.getCity(),
            owner.getTelephone(),
            owner.getEmail(),
            pets
        );
    }

    private Owner toEntity(OwnerDto dto) {
        Owner owner = new Owner();
        owner.setFirstName(dto.firstName());
        owner.setLastName(dto.lastName());
        owner.setAddress(dto.address());
        owner.setCity(dto.city());
        owner.setTelephone(dto.telephone());
        owner.setEmail(dto.email());
        return owner;
    }

    private void updateEntity(Owner owner, OwnerDto dto) {
        owner.setFirstName(dto.firstName());
        owner.setLastName(dto.lastName());
        owner.setAddress(dto.address());
        owner.setCity(dto.city());
        owner.setTelephone(dto.telephone());
        owner.setEmail(dto.email());
    }
}
